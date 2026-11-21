package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

type deviceAuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type deviceEventRequest struct {
	DeviceID         int64           `json:"device_id"`
	EventType        string          `json:"event_type"`
	IdempotencyKey   string          `json:"idempotency_key"`
	MemberIdentifier string          `json:"member_identifier"`
	EventTimestamp   string          `json:"event_timestamp"`
	HMACSignature    string          `json:"hmac_signature"`
	Payload          json.RawMessage `json:"payload"`
}

func (s *Server) deviceSecret() []byte {
	if s.Config.DeviceHMACSecret != "" {
		return []byte(s.Config.DeviceHMACSecret)
	}
	return []byte(s.Config.SessionSecret)
}

func (s *Server) issueDeviceToken(deviceID int64, tokenVersion int) string {
	expires := time.Now().Add(time.Hour).Unix()
	payload := fmt.Sprintf("%d|%d|%d", deviceID, expires, tokenVersion)
	mac := hmac.New(sha256.New, s.deviceSecret())
	mac.Write([]byte(payload))
	signature := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(signature)
}

func (s *Server) parseDeviceToken(token string) (int64, int, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return 0, 0, false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, 0, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, 0, false
	}
	mac := hmac.New(sha256.New, s.deviceSecret())
	mac.Write(payloadBytes)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return 0, 0, false
	}
	values := strings.Split(string(payloadBytes), "|")
	if len(values) != 2 && len(values) != 3 {
		return 0, 0, false
	}
	deviceID, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	expires, err := strconv.ParseInt(values[1], 10, 64)
	if err != nil || time.Now().Unix() > expires {
		return 0, 0, false
	}
	// Older two-part tokens are treated as version 0 for compatibility.
	version := 0
	if len(values) == 3 {
		version, err = strconv.Atoi(values[2])
		if err != nil || version < 0 {
			return 0, 0, false
		}
	}
	return deviceID, version, true
}

func (s *Server) authenticateDevice(c *gin.Context) (*models.DeviceRegistration, bool) {
	token := strings.TrimSpace(c.GetHeader("X-Device-Token"))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-Device-Api-Key"))
	}
	if token == "" {
		return nil, false
	}

	var device models.DeviceRegistration
	var err error
	if deviceID, tokenVersion, ok := s.parseDeviceToken(token); ok {
		err = s.DB.WithContext(c.Request.Context()).First(&device, deviceID).Error
		if err == nil && tokenVersion != device.TokenVersion {
			return nil, false
		}
	} else {
		err = s.DB.WithContext(c.Request.Context()).Where("api_key_digest = ?", hashDeviceAPIKey(token)).First(&device).Error
	}
	if err != nil {
		return nil, false
	}
	if device.Status != "active" {
		return nil, false
	}
	now := time.Now().UTC()
	s.DB.WithContext(c.Request.Context()).Model(&models.DeviceRegistration{}).Where("id = ?", device.ID).Update("last_seen_at", now)
	device.LastSeenAt = &now
	return &device, true
}

func (s *Server) authenticateDeviceOrFail(c *gin.Context) (*models.DeviceRegistration, bool) {
	device, ok := s.authenticateDevice(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return nil, false
	}
	return device, true
}

func verifyDeviceEventSignature(secret []byte, request deviceEventRequest, signature string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	canonical := fmt.Sprintf("%d|%s|%s|%s|%s",
		request.DeviceID,
		strings.TrimSpace(request.EventType),
		strings.TrimSpace(request.IdempotencyKey),
		strings.TrimSpace(request.MemberIdentifier),
		strings.TrimSpace(request.EventTimestamp),
	)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(canonical))
	expected := mac.Sum(nil)

	provided, err := hex.DecodeString(signature)
	if err != nil {
		provided, err = base64.StdEncoding.DecodeString(signature)
	}
	if err != nil {
		provided, err = base64.RawStdEncoding.DecodeString(signature)
	}
	if err != nil {
		return false
	}
	return hmac.Equal(expected, provided)
}

func (s *Server) authenticateDeviceEndpoint(c *gin.Context) {
	var request struct {
		APIKey     string `json:"api_key"`
		DeviceName string `json:"device_name"`
	}
	if !bindJSON(c, &request) {
		return
	}
	apiKey := strings.TrimSpace(request.APIKey)
	if apiKey == "" {
		fail(c, http.StatusUnauthorized, "API key is required")
		return
	}
	var device models.DeviceRegistration
	if err := s.DB.WithContext(c.Request.Context()).Where("api_key_digest = ?", hashDeviceAPIKey(apiKey)).First(&device).Error; err != nil {
		fail(c, http.StatusUnauthorized, "Invalid device credentials")
		return
	}
	if device.Status != "active" {
		fail(c, http.StatusForbidden, "Device is not active")
		return
	}
	now := time.Now().UTC()
	updates := map[string]any{"last_seen_at": now}
	if name := strings.TrimSpace(request.DeviceName); name != "" {
		updates["name"] = name
	}
	s.DB.WithContext(c.Request.Context()).Model(&models.DeviceRegistration{}).Where("id = ?", device.ID).Updates(updates)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device authenticated successfully",
		"device": gin.H{
			"id":          device.ID,
			"device_uid":  device.DeviceUID,
			"name":        device.Name,
			"device_type": device.DeviceType,
			"status":      device.Status,
			"token":       s.issueDeviceToken(device.ID, device.TokenVersion),
			"expires_in":  3600,
		},
	})
}

func (s *Server) enrollFingerprint(c *gin.Context) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return
	}
	var request struct {
		MemberID            int64  `json:"member_id"`
		FingerprintTemplate string `json:"fingerprint_template"`
		DeviceUserID        string `json:"device_user_id"`
	}
	if !bindJSON(c, &request) {
		return
	}
	request.FingerprintTemplate = strings.TrimSpace(request.FingerprintTemplate)
	if request.FingerprintTemplate == "" {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint template is required")
		return
	}
	// Raw fingerprint data is never persisted. Only the device-side template reference is stored.
	if request.MemberID <= 0 {
		fail(c, http.StatusUnprocessableEntity, "Member is required")
		return
	}
	var member models.Member
	err := s.DB.WithContext(c.Request.Context()).
		Where("id = ? AND status = ? AND deleted_at IS NULL", request.MemberID, "active").
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, "Member not found or inactive")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to enroll fingerprint")
		return
	}

	deviceUserID := strings.TrimSpace(request.DeviceUserID)
	if deviceUserID == "" {
		buffer := make([]byte, 8)
		if _, err := rand.Read(buffer); err != nil {
			fail(c, http.StatusInternalServerError, "Unable to enroll fingerprint")
			return
		}
		deviceUserID = hex.EncodeToString(buffer)
	}

	var enrollment models.BiometricEnrollment
	err = s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("member_id = ? AND device_registration_id = ?", member.ID, device.ID).First(&enrollment).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			enrollment = models.BiometricEnrollment{
				MemberID:             member.ID,
				DeviceRegistrationID: device.ID,
				DeviceUserID:         deviceUserID,
				Status:               "enrolled",
			}
			return tx.Create(&enrollment).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&models.BiometricEnrollment{}).Where("id = ?", enrollment.ID).Updates(map[string]any{
			"device_user_id": deviceUserID,
			"status":         "enrolled",
		}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to enroll fingerprint")
		return
	}

	s.writeDeviceAudit(c, "biometric_enrollment_confirmed", "BiometricEnrollment", &enrollment.ID, gin.H{
		"member_id":      member.ID,
		"device_id":      device.ID,
		"device_user_id": deviceUserID,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Fingerprint enrolled successfully",
		"enrollment": gin.H{
			"id":              enrollment.ID,
			"member_id":       enrollment.MemberID,
			"member_name":     member.Name,
			"member_login_id": member.LoginID,
			"device_user_id":  deviceUserID,
			"status":          "enrolled",
		},
	})
}

func (s *Server) listBiometricEnrollments(c *gin.Context) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return
	}
	var enrollments []models.BiometricEnrollment
	if err := s.DB.WithContext(c.Request.Context()).
		Where("device_registration_id = ?", device.ID).
		Order("updated_at DESC").Find(&enrollments).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load enrollments")
		return
	}
	items := make([]gin.H, 0, len(enrollments))
	for i := range enrollments {
		var member models.Member
		if err := s.DB.WithContext(c.Request.Context()).First(&member, enrollments[i].MemberID).Error; err != nil {
			continue
		}
		items = append(items, gin.H{
			"id":               enrollments[i].ID,
			"member_id":        enrollments[i].MemberID,
			"member_name":      member.Name,
			"member_login_id":  member.LoginID,
			"member_usn":       member.USN,
			"reader_key":       enrollments[i].ReaderKey,
			"device_user_id":   enrollments[i].DeviceUserID,
			"fingerprint_slot": enrollments[i].FingerprintSlot,
			"status":           enrollments[i].Status,
			"enrolled_at":      enrollments[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "enrollments": items})
}

func (s *Server) removeBiometricEnrollment(c *gin.Context) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var enrollment models.BiometricEnrollment
	if err := s.DB.WithContext(c.Request.Context()).First(&enrollment, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	if enrollment.DeviceRegistrationID != device.ID {
		fail(c, http.StatusForbidden, "This enrollment does not belong to this device")
		return
	}
	var member models.Member
	s.DB.WithContext(c.Request.Context()).First(&member, enrollment.MemberID)

	if err := s.DB.WithContext(c.Request.Context()).Delete(&models.BiometricEnrollment{}, id).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to remove enrollment")
		return
	}
	s.writeDeviceAudit(c, "biometric_enrollment_removed", "BiometricEnrollment", &enrollment.ID, gin.H{
		"member_id": member.ID,
		"device_id": device.ID,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Fingerprint enrollment removed successfully",
		"member":  gin.H{"id": member.ID, "login_id": member.LoginID, "name": member.Name},
	})
}

func (s *Server) biometricDeviceStatus(c *gin.Context) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return
	}
	var pending int64
	s.DB.WithContext(c.Request.Context()).Model(&models.DeviceEvent{}).
		Where("device_registration_id = ? AND processing_status = ?", device.ID, "pending").Count(&pending)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"device": gin.H{
			"id":             device.ID,
			"name":           device.Name,
			"device_type":    device.DeviceType,
			"status":         device.Status,
			"last_seen_at":   device.LastSeenAt,
			"pending_events": pending,
		},
	})
}

func (s *Server) submitBiometricEvent(c *gin.Context) {
	var request deviceEventRequest
	if !bindJSON(c, &request) {
		return
	}
	request.EventType = strings.TrimSpace(request.EventType)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.MemberIdentifier = strings.TrimSpace(request.MemberIdentifier)
	if request.DeviceID <= 0 || request.EventType == "" || request.IdempotencyKey == "" {
		fail(c, http.StatusUnprocessableEntity, "Device ID, event type, and idempotency key are required")
		return
	}
	allowed := map[string]bool{
		"fingerprint_scan": true, "finger_mismatch": true,
		"enrollment_success": true, "enrollment_failure": true,
	}
	if !allowed[request.EventType] {
		fail(c, http.StatusUnprocessableEntity, "Invalid event type")
		return
	}

	device := currentDevice(c)
	if device != nil && device.ID != request.DeviceID {
		fail(c, http.StatusForbidden, "Device token does not match device ID")
		return
	}
	if device == nil {
		if !verifyDeviceEventSignature(s.deviceSecret(), request, request.HMACSignature) {
			fail(c, http.StatusUnauthorized, "Device authentication is required")
			return
		}
		var stored models.DeviceRegistration
		if err := s.DB.WithContext(c.Request.Context()).First(&stored, request.DeviceID).Error; err != nil || stored.Status != "active" {
			fail(c, http.StatusForbidden, "Device is not active")
			return
		}
		device = &stored
	}

	var existing models.DeviceEvent
	err := s.DB.WithContext(c.Request.Context()).Where("idempotency_key = ?", request.IdempotencyKey).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Event already processed",
			"event":   serializeDeviceEvent(&existing),
		})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusInternalServerError, "Unable to record event")
		return
	}

	payload := request.Payload
	if len(payload) == 0 || !json.Valid(payload) {
		payload = json.RawMessage(`{}`)
	}
	eventTimestamp := parseDeviceTimestamp(request.EventTimestamp)
	event := models.DeviceEvent{
		DeviceRegistrationID: device.ID,
		EventType:            request.EventType,
		IdempotencyKey:       request.IdempotencyKey,
		MemberIdentifier:     nullableString(request.MemberIdentifier),
		HMACSignature:        nullableString(request.HMACSignature),
		EventTimestamp:       eventTimestamp,
		Payload:              payload,
		ProcessingStatus:     "pending",
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&event).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			s.DB.WithContext(c.Request.Context()).Where("idempotency_key = ?", request.IdempotencyKey).First(&existing)
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Event already processed", "event": serializeDeviceEvent(&existing)})
			return
		}
		fail(c, http.StatusInternalServerError, "Unable to record event")
		return
	}
	now := time.Now().UTC()
	s.DB.WithContext(c.Request.Context()).Model(&models.DeviceRegistration{}).Where("id = ?", device.ID).Update("last_seen_at", now)

	jobPayload, _ := json.Marshal(map[string]int64{"event_id": event.ID})
	if s.Jobs != nil {
		if err := s.Jobs.Enqueue(c.Request.Context(), "biometric.process_event", jobPayload, time.Now().UTC()); err != nil {
			s.Log.Error("failed to enqueue biometric event processing", "event_id", event.ID, "error", err)
		}
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Event received and queued for processing",
		"event":   serializeDeviceEvent(&event),
	})
}

func (s *Server) syncBiometricEvents(c *gin.Context) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return
	}
	var request struct {
		DeviceID          int64  `json:"device_id"`
		LastSyncTimestamp string `json:"last_sync_timestamp"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if request.DeviceID <= 0 {
		fail(c, http.StatusUnprocessableEntity, "Device ID is required")
		return
	}
	if request.DeviceID != device.ID {
		fail(c, http.StatusForbidden, "Device token does not match device ID")
		return
	}

	query := s.DB.WithContext(c.Request.Context()).Where("device_registration_id = ?", device.ID)
	if lastSync, ok := parseDate(request.LastSyncTimestamp); ok {
		query = query.Where("created_at > ?", lastSync)
	}
	var events []models.DeviceEvent
	if err := query.Order("created_at ASC").Limit(100).Find(&events).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to sync events")
		return
	}
	items := make([]gin.H, 0, len(events))
	for i := range events {
		items = append(items, serializeDeviceEvent(&events[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"events":  items,
		"sync_metadata": gin.H{
			"last_sync_timestamp": time.Now().UTC(),
			"events_synced":       len(events),
			"has_more":            len(events) >= 100,
		},
	})
}

func serializeDeviceEvent(event *models.DeviceEvent) gin.H {
	var payload any = map[string]any{}
	if len(event.Payload) > 0 {
		_ = json.Unmarshal(event.Payload, &payload)
	}
	return gin.H{
		"id":                     event.ID,
		"device_registration_id": event.DeviceRegistrationID,
		"event_type":             event.EventType,
		"idempotency_key":        event.IdempotencyKey,
		"member_identifier":      event.MemberIdentifier,
		"event_timestamp":        event.EventTimestamp,
		"payload":                payload,
		"processing_status":      event.ProcessingStatus,
		"created_at":             event.CreatedAt,
	}
}

func parseDeviceTimestamp(value string) time.Time {
	now := time.Now().UTC()
	parsed, ok := parseDate(value)
	if !ok {
		return now
	}
	parsed = parsed.UTC()
	if parsed.Before(now.Add(-5 * time.Minute)) {
		return now.Add(-5 * time.Minute)
	}
	if parsed.After(now.Add(5 * time.Minute)) {
		return now.Add(5 * time.Minute)
	}
	return parsed
}

func (s *Server) writeDeviceAudit(c *gin.Context, action, resourceType string, resourceID *int64, details gin.H) {
	encoded, _ := json.Marshal(details)
	ip := c.ClientIP()
	entry := models.AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      encoded,
		IPAddress:    &ip,
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&entry).Error; err != nil {
		s.Log.Error("device audit log write failed", "action", action, "error", err)
	}
}

func (s *Server) processBiometricEvent(ctx context.Context, payload json.RawMessage) error {
	var request struct {
		EventID int64 `json:"event_id"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return err
	}
	var event models.DeviceEvent
	if err := s.DB.WithContext(ctx).First(&event, request.EventID).Error; err != nil {
		return err
	}
	if event.ProcessingStatus == "completed" {
		return nil
	}

	switch event.EventType {
	case "fingerprint_scan":
		if err := s.processFingerprintScan(ctx, &event); err != nil {
			s.DB.WithContext(ctx).Model(&models.DeviceEvent{}).Where("id = ?", event.ID).Updates(map[string]any{
				"processing_status": "failed",
				"processing_error":  err.Error(),
			})
			return err
		}
	}

	return s.DB.WithContext(ctx).Model(&models.DeviceEvent{}).Where("id = ?", event.ID).Updates(map[string]any{
		"processing_status": "completed",
		"processing_error":  nil,
	}).Error
}

func (s *Server) processFingerprintScan(ctx context.Context, event *models.DeviceEvent) error {
	if event.MemberIdentifier == nil || strings.TrimSpace(*event.MemberIdentifier) == "" {
		return nil
	}
	identifier := strings.TrimSpace(*event.MemberIdentifier)
	var member models.Member
	err := s.DB.WithContext(ctx).
		Where("(login_id = ? OR usn = ? OR roll_number = ?) AND status = ? AND deleted_at IS NULL", identifier, strings.ToUpper(identifier), identifier, "active").
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.Log.Warn("biometric scan for unknown member identifier", "identifier", identifier)
		return nil
	}
	if err != nil {
		return err
	}

	now := time.Now()
	result, err := s.applyAttendanceRecord(
		s.DB.WithContext(ctx),
		&member,
		now,
		now.UTC(),
		nullableString(fmt.Sprintf("%d", event.ID)),
		0,
	)
	if err != nil {
		return err
	}
	switch result.Outcome {
	case attendanceOutcomeEntry:
		s.writeSystemAudit(ctx, "attendance_login_biometric", "AttendanceLog", &result.Log.ID, member, event)
	case attendanceOutcomeExit:
		s.writeSystemAudit(ctx, "attendance_logout_biometric", "AttendanceLog", &result.Log.ID, member, event)
	}
	return nil
}

type attendanceApplicationResult struct {
	Outcome string
	Log     *models.AttendanceLog
}

// applyAttendanceRecord is the single source of truth for the Entry/Exit
// toggle. The existing door path passes a zero dedupe window and its historical
// event-id device reference. Attendance terminals pass their own reference and
// a duplicate window so retries and double scans do not close a fresh entry.
func (s *Server) applyAttendanceRecord(
	db *gorm.DB,
	member *models.Member,
	entryAt time.Time,
	exitAt time.Time,
	deviceReference *string,
	dedupeWindow time.Duration,
) (*attendanceApplicationResult, error) {
	today := startOfDay(entryAt)
	var last models.AttendanceLog
	err := db.
		Where("member_id = ? AND entry_date = ?", member.ID, today).
		Order("entry_time DESC").First(&last).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if dedupeWindow > 0 && err == nil {
		duplicate := false
		if last.ExitTime == nil {
			duplicate = !entryAt.After(last.EntryTime) || entryAt.Sub(last.EntryTime) < dedupeWindow
		} else {
			duplicate = !entryAt.After(*last.ExitTime) || entryAt.Sub(*last.ExitTime) < dedupeWindow
		}
		if duplicate {
			return &attendanceApplicationResult{Outcome: attendanceOutcomeDuplicate, Log: &last}, nil
		}
	}

	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && last.ExitTime != nil) {
		record := models.AttendanceLog{
			MemberID:        member.ID,
			EntryDate:       today,
			EntryTime:       entryAt,
			Source:          "biometric",
			DeviceReference: deviceReference,
		}
		if err := db.Create(&record).Error; err != nil {
			return nil, err
		}
		return &attendanceApplicationResult{Outcome: attendanceOutcomeEntry, Log: &record}, nil
	}
	if err != nil {
		return nil, err
	}
	if err := db.Model(&models.AttendanceLog{}).Where("id = ?", last.ID).Update("exit_time", exitAt).Error; err != nil {
		return nil, err
	}
	last.ExitTime = &exitAt
	return &attendanceApplicationResult{Outcome: attendanceOutcomeExit, Log: &last}, nil
}

func (s *Server) writeSystemAudit(ctx context.Context, action, resourceType string, resourceID *int64, member models.Member, event *models.DeviceEvent) {
	encoded, _ := json.Marshal(gin.H{
		"member_login_id": member.LoginID,
		"member_name":     member.Name,
		"device_id":       event.DeviceRegistrationID,
		"device_event_id": event.ID,
	})
	entry := models.AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      encoded,
	}
	if err := s.DB.WithContext(ctx).Create(&entry).Error; err != nil {
		s.Log.Error("biometric audit write failed", "action", action, "error", err)
	}
}
