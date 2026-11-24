package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const (
	doorAccessDedupeWindow = 3 * time.Second
	doorAccessUnlockMS     = 5000
)

type doorAccessRequest struct {
	FingerprintSlot *int   `json:"fingerprint_slot"`
	ReaderKey       string `json:"reader_key"`
}

func (s *Server) registerDoorAccessRoutes(api *gin.RouterGroup) {
	api.POST("/biometric/door-access", s.RequireDevice(), s.authorizeDoorAccess)
}

func (s *Server) doorAccessDenied(c *gin.Context, reason string) {
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"authorized": false,
		"duplicate":  false,
		"reason":     reason,
	})
}

func (s *Server) authorizeDoorAccess(c *gin.Context) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return
	}

	var request doorAccessRequest
	if !bindJSON(c, &request) {
		return
	}
	if request.FingerprintSlot == nil || !validateEnrollmentSlot(*request.FingerprintSlot) {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint slot must be between 1 and 1000")
		return
	}
	readerKey := normalizeEnrollmentReaderKey(request.ReaderKey)
	if readerKey == "" {
		readerKey = enrollmentReaderLogin
	}
	if readerKey != enrollmentReaderLogin {
		s.doorAccessDenied(c, "reader_not_authorized")
		return
	}

	ctx := c.Request.Context()
	var enrollment models.BiometricEnrollment
	err := s.DB.WithContext(ctx).
		Where("device_registration_id = ? AND fingerprint_slot = ? AND status IN ?",
			device.ID, *request.FingerprintSlot, []string{"enrolled", "verified"}).
		First(&enrollment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.doorAccessDenied(c, "fingerprint_not_enrolled")
		return
	}
	if err != nil {
		s.Log.Error("door access enrollment lookup failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to authorize door access")
		return
	}
	if enrollment.ReaderKey != nil && strings.TrimSpace(*enrollment.ReaderKey) != "" &&
		normalizeEnrollmentReaderKey(*enrollment.ReaderKey) != enrollmentReaderLogin {
		s.doorAccessDenied(c, "reader_not_authorized")
		return
	}

	var member models.Member
	err = s.DB.WithContext(ctx).
		Where("id = ? AND status = ? AND deleted_at IS NULL", enrollment.MemberID, "active").
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.doorAccessDenied(c, "member_not_active")
		return
	}
	if err != nil {
		s.Log.Error("door access member lookup failed", "member_id", enrollment.MemberID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to authorize door access")
		return
	}

	var user models.User
	if err := s.DB.WithContext(ctx).First(&user, member.UserID).Error; err != nil || !user.Active {
		s.doorAccessDenied(c, "member_not_active")
		return
	}

	now := time.Now().UTC()
	var recent int64
	if err := s.DB.WithContext(ctx).Model(&models.AuditLog{}).
		Where("action = ? AND resource_type = ? AND resource_id = ? AND created_at > ?",
			"door_access_granted", "Member", member.ID, now.Add(-doorAccessDedupeWindow)).
		Count(&recent).Error; err != nil {
		s.Log.Error("door access duplicate check failed", "member_id", member.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to authorize door access")
		return
	}
	if recent > 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"authorized":     true,
			"duplicate":      true,
			"reason":         "duplicate_scan",
			"unlock_seconds": doorAccessUnlockMS / 1000,
			"member": gin.H{
				"id":       member.ID,
				"name":     member.Name,
				"login_id": member.LoginID,
			},
		})
		return
	}

	s.writeDeviceAudit(c, "door_access_granted", "Member", &member.ID, gin.H{
		"device_id":        device.ID,
		"device_uid":       device.DeviceUID,
		"fingerprint_slot": enrollment.FingerprintSlot,
		"reader_key":       readerKey,
		"source":           "door_access",
	})
	s.enqueueDoorAccessAttendance(ctx, device, &member, *request.FingerprintSlot, readerKey)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"authorized":     true,
		"duplicate":      false,
		"reason":         "authorized",
		"unlock_seconds": doorAccessUnlockMS / 1000,
		"member": gin.H{
			"id":       member.ID,
			"name":     member.Name,
			"login_id": member.LoginID,
		},
	})
}

func (s *Server) enqueueDoorAccessAttendance(ctx context.Context, device *models.DeviceRegistration, member *models.Member, slot int, readerKey string) {
	payload, err := json.Marshal(gin.H{
		"source":           "door_access",
		"fingerprint_slot": slot,
		"reader_key":       readerKey,
	})
	if err != nil {
		s.Log.Error("door access attendance payload failed", "member_id", member.ID, "error", err)
		return
	}
	event := models.DeviceEvent{
		DeviceRegistrationID: device.ID,
		EventType:            "fingerprint_scan",
		IdempotencyKey:       fmt.Sprintf("door-%d-%d-%d-%d", device.ID, member.ID, slot, time.Now().UTC().UnixNano()),
		MemberIdentifier:     stringPointer(member.LoginID),
		EventTimestamp:       time.Now().UTC(),
		Payload:              payload,
		ProcessingStatus:     "pending",
	}
	if err := s.DB.WithContext(ctx).Create(&event).Error; err != nil {
		s.Log.Error("door access attendance event failed", "member_id", member.ID, "error", err)
		return
	}

	jobPayload, err := json.Marshal(map[string]int64{"event_id": event.ID})
	if err != nil {
		s.Log.Error("door access attendance job payload failed", "event_id", event.ID, "error", err)
		return
	}
	if s.Jobs != nil {
		if err := s.Jobs.Enqueue(ctx, "biometric.process_event", jobPayload, time.Now().UTC()); err != nil {
			s.Log.Error("door access attendance enqueue failed", "event_id", event.ID, "error", err)
		}
		return
	}
	if err := s.processBiometricEvent(ctx, jobPayload); err != nil {
		s.Log.Error("door access attendance processing failed", "event_id", event.ID, "error", err)
	}
}
