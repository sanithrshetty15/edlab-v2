package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

var (
	errAttendanceNonAttendanceMapping = errors.New("member already has a non-attendance mapping on this terminal")
	errAttendanceMemberSlotConflict   = errors.New("member already has a different fingerprint slot on this terminal")
	errAttendanceSlotTaken            = errors.New("fingerprint slot is already assigned to another member")
)

var attendanceProgressStatuses = map[string]bool{
	attendanceSessionStatusInProgress: true,
	"waiting_for_finger":              true,
	"finger_detected":                 true,
	"captured":                        true,
	attendanceSessionStatusFailed:     true,
}

func serializeAttendanceEnrollmentSession(session *models.AttendanceEnrollmentSession, member *models.Member, device *models.DeviceRegistration) gin.H {
	data := gin.H{
		"id":                     session.ID,
		"session_uid":            session.SessionUID,
		"member_id":              session.MemberID,
		"device_registration_id": session.DeviceRegistrationID,
		"operation":              session.Operation,
		"reader_key":             session.ReaderKey,
		"status":                 session.Status,
		"fingerprint_slot":       session.FingerprintSlot,
		"device_user_id":         session.DeviceUserID,
		"attempts":               session.Attempts,
		"error_code":             session.ErrorCode,
		"error_message":          session.ErrorMessage,
		"cleanup_required":       session.CleanupRequired,
		"is_active":              session.IsActive,
		"claimed_at":             session.ClaimedAt,
		"claimed_by_device_id":   session.ClaimedByDeviceID,
		"expires_at":             session.ExpiresAt,
		"started_at":             session.StartedAt,
		"completed_at":           session.CompletedAt,
		"cleanup_completed_at":   session.CleanupCompletedAt,
		"created_at":             session.CreatedAt,
		"updated_at":             session.UpdatedAt,
	}
	if session.ReaderKey != nil {
		data["reader_label"] = enrollmentReaderLabel(*session.ReaderKey)
	}
	if member != nil {
		data["member_name"] = member.Name
		data["member_login_id"] = member.LoginID
	}
	if device != nil {
		data["device"] = serializeDevice(device)
	}
	return data
}

func (s *Server) loadAttendanceDeviceSession(c *gin.Context) (*models.DeviceRegistration, *models.AttendanceEnrollmentSession, bool) {
	device, ok := attendanceTerminalFromContext(c)
	if !ok {
		return nil, nil, false
	}
	uid := strings.TrimSpace(c.Param("uid"))
	if uid == "" {
		fail(c, http.StatusNotFound, "Attendance enrollment session not found")
		return nil, nil, false
	}
	var session models.AttendanceEnrollmentSession
	if err := s.DB.WithContext(c.Request.Context()).Where("session_uid = ?", uid).First(&session).Error; err != nil {
		fail(c, http.StatusNotFound, "Attendance enrollment session not found")
		return nil, nil, false
	}
	if session.DeviceRegistrationID != device.ID {
		fail(c, http.StatusForbidden, "This attendance enrollment session does not belong to this device")
		return nil, nil, false
	}
	return device, &session, true
}

func expireAttendanceSession(tx *gorm.DB, session *models.AttendanceEnrollmentSession, now time.Time) error {
	cleanupRequired := session.CleanupRequired || (session.Operation == attendanceSessionOperationEnroll && session.StartedAt != nil)
	updates := map[string]any{
		"status":           attendanceSessionStatusExpired,
		"is_active":        false,
		"completed_at":     now,
		"cleanup_required": cleanupRequired,
		"updated_at":       now,
	}
	if err := tx.Model(&models.AttendanceEnrollmentSession{}).Where("id = ? AND is_active = true", session.ID).Updates(updates).Error; err != nil {
		return err
	}
	session.Status = attendanceSessionStatusExpired
	session.IsActive = false
	session.CompletedAt = &now
	session.CleanupRequired = cleanupRequired
	session.UpdatedAt = now
	return nil
}

func (s *Server) claimAttendanceEnrollmentSession(c *gin.Context) {
	device, ok := attendanceTerminalFromContext(c)
	if !ok {
		return
	}
	if !s.allowAttendanceDeviceRequest(c, device, "enrollment-poll", attendancePollRateLimit, time.Minute) {
		return
	}
	ctx := c.Request.Context()
	now := time.Now().UTC()

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cleanup models.AttendanceEnrollmentSession
		if err := tx.Where("device_registration_id = ? AND cleanup_required = true AND cleanup_completed_at IS NULL", device.ID).
			Order("id ASC").First(&cleanup).Error; err == nil {
			c.JSON(http.StatusOK, gin.H{
				"success":     true,
				"action":      "cleanup",
				"enrollment":  serializeAttendanceEnrollmentSession(&cleanup, nil, device),
				"server_time": now,
			})
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var session models.AttendanceEnrollmentSession
		err := tx.Where("device_registration_id = ? AND is_active = true AND (claimed_by_device_id IS NULL OR claimed_by_device_id = ?)", device.ID, device.ID).
			Order("id ASC").First(&session).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"success":     true,
				"action":      "none",
				"enrollment":  nil,
				"server_time": now,
			})
			return nil
		}
		if err != nil {
			return err
		}
		if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
			if err := expireAttendanceSession(tx, &session, now); err != nil {
				return err
			}
			if session.CleanupRequired {
				c.JSON(http.StatusOK, gin.H{
					"success":     true,
					"action":      "cleanup",
					"enrollment":  serializeAttendanceEnrollmentSession(&session, nil, device),
					"server_time": now,
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"success":     true,
					"action":      "none",
					"enrollment":  nil,
					"server_time": now,
				})
			}
			return nil
		}

		var member *models.Member
		var loaded models.Member
		if err := tx.First(&loaded, session.MemberID).Error; err == nil {
			member = &loaded
		}
		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"action":      session.Operation,
			"enrollment":  serializeAttendanceEnrollmentSession(&session, member, device),
			"server_time": now,
		})
		return nil
	})
	if err != nil {
		s.Log.Error("attendance enrollment poll failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance enrollment session")
	}
}

func (s *Server) showAttendanceEnrollmentSessionForDevice(c *gin.Context) {
	device, session, ok := s.loadAttendanceDeviceSession(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	now := time.Now().UTC()
	if session.IsActive && session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := expireAttendanceSession(s.DB.WithContext(ctx), session, now); err != nil {
			s.Log.Error("attendance session expiry failed", "session_id", session.ID, "error", err)
		}
	}
	action := "none"
	switch {
	case session.CleanupRequired && session.CleanupCompletedAt == nil:
		action = "cleanup"
	case session.IsActive:
		action = session.Operation
	case session.Status == attendanceSessionStatusCompleted:
		action = "completed"
	}
	var member *models.Member
	var loaded models.Member
	if err := s.DB.WithContext(ctx).First(&loaded, session.MemberID).Error; err == nil {
		member = &loaded
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"action":      action,
		"enrollment":  serializeAttendanceEnrollmentSession(session, member, device),
		"server_time": now,
	})
}

func (s *Server) claimAttendanceEnrollmentCommand(c *gin.Context) {
	device, session, ok := s.loadAttendanceDeviceSession(c)
	if !ok {
		return
	}
	now := time.Now().UTC()
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := expireAttendanceSession(s.DB.WithContext(c.Request.Context()), session, now); err != nil {
			s.Log.Error("attendance session expiry failed", "session_id", session.ID, "error", err)
		}
		fail(c, http.StatusConflict, "Attendance enrollment session expired")
		return
	}
	if !session.IsActive || session.Status == attendanceSessionStatusCompleted ||
		session.Status == attendanceSessionStatusFailed || session.Status == attendanceSessionStatusCancelled ||
		session.Status == attendanceSessionStatusExpired {
		fail(c, http.StatusConflict, "Attendance enrollment is no longer active")
		return
	}
	if session.CleanupRequired {
		fail(c, http.StatusConflict, "Attendance enrollment requires cleanup first")
		return
	}
	if session.ReaderKey == nil || *session.ReaderKey == "" {
		fail(c, http.StatusConflict, "Attendance enrollment has no selected reader")
		return
	}
	if session.ClaimedByDeviceID != nil && *session.ClaimedByDeviceID != device.ID {
		fail(c, http.StatusConflict, "Attendance enrollment command is owned by another device")
		return
	}
	if session.ClaimedAt == nil {
		result := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceEnrollmentSession{}).
			Where("id = ? AND is_active = true AND (claimed_by_device_id IS NULL OR claimed_by_device_id = ?)", session.ID, device.ID).
			Updates(map[string]any{
				"claimed_at":           now,
				"claimed_by_device_id": device.ID,
				"started_at":           now,
				"updated_at":           now,
			})
		if result.Error != nil {
			s.Log.Error("attendance enrollment claim failed", "session_id", session.ID, "error", result.Error)
			fail(c, http.StatusInternalServerError, "Unable to claim attendance enrollment command")
			return
		}
		if result.RowsAffected == 0 {
			fail(c, http.StatusConflict, "Attendance enrollment command was claimed by another device")
			return
		}
		session.ClaimedAt = &now
		session.ClaimedByDeviceID = &device.ID
		if session.StartedAt == nil {
			session.StartedAt = &now
		}
	}
	s.writeDeviceAudit(c, "attendance_enrollment_claimed", "AttendanceEnrollmentSession", &session.ID, gin.H{
		"device_id":   device.ID,
		"session_uid": session.SessionUID,
		"operation":   session.Operation,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Attendance enrollment command acknowledged",
		"enrollment": serializeAttendanceEnrollmentSession(session, nil, device),
	})
}

func (s *Server) updateAttendanceEnrollmentProgress(c *gin.Context) {
	device, session, ok := s.loadAttendanceDeviceSession(c)
	if !ok {
		return
	}
	var request struct {
		Status          string `json:"status"`
		FingerprintSlot *int   `json:"fingerprint_slot"`
		ErrorCode       string `json:"error_code"`
		ErrorMessage    string `json:"error_message"`
	}
	if !bindJSON(c, &request) {
		return
	}
	status := strings.ToLower(strings.TrimSpace(request.Status))
	if status == "" {
		fail(c, http.StatusUnprocessableEntity, "Enrollment status is required")
		return
	}
	if !attendanceProgressStatuses[status] {
		fail(c, http.StatusUnprocessableEntity, "Invalid enrollment status")
		return
	}
	if request.FingerprintSlot != nil && !validateEnrollmentSlot(*request.FingerprintSlot) {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint slot must be between 1 and 127")
		return
	}
	now := time.Now().UTC()
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := expireAttendanceSession(s.DB.WithContext(c.Request.Context()), session, now); err != nil {
			s.Log.Error("attendance session expiry failed", "session_id", session.ID, "error", err)
		}
		fail(c, http.StatusConflict, "Attendance enrollment session expired")
		return
	}
	if !session.IsActive {
		fail(c, http.StatusConflict, "Attendance enrollment is no longer active")
		return
	}
	if status == attendanceSessionStatusFailed {
		s.applyAttendanceEnrollmentFailure(c, device, session, request.FingerprintSlot, request.ErrorCode, request.ErrorMessage)
		return
	}
	if request.FingerprintSlot != nil {
		if session.FingerprintSlot != nil && *session.FingerprintSlot != *request.FingerprintSlot {
			fail(c, http.StatusConflict, "Fingerprint slot does not match the slot already reported for this session")
			return
		}
	}
	updates := map[string]any{
		"status":        status,
		"attempts":      gorm.Expr("attempts + 1"),
		"error_code":    nil,
		"error_message": nil,
		"updated_at":    now,
	}
	if session.StartedAt == nil {
		updates["started_at"] = now
	}
	if request.FingerprintSlot != nil {
		updates["fingerprint_slot"] = *request.FingerprintSlot
	}
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceEnrollmentSession{}).
		Where("id = ? AND is_active = true", session.ID).Updates(updates).Error; err != nil {
		s.Log.Error("attendance enrollment progress failed", "session_id", session.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to update attendance enrollment")
		return
	}
	var updated models.AttendanceEnrollmentSession
	if err := s.DB.WithContext(c.Request.Context()).First(&updated, session.ID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load attendance enrollment")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Attendance enrollment progress recorded",
		"enrollment": serializeAttendanceEnrollmentSession(&updated, nil, device),
	})
}

func (s *Server) applyAttendanceEnrollmentFailure(c *gin.Context, device *models.DeviceRegistration, session *models.AttendanceEnrollmentSession, slot *int, code, message string) {
	now := time.Now().UTC()
	if session.Status == attendanceSessionStatusCompleted {
		fail(c, http.StatusConflict, "Attendance enrollment is already completed")
		return
	}
	if !session.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "Attendance enrollment is already finished",
			"enrollment": serializeAttendanceEnrollmentSession(session, nil, device),
		})
		return
	}
	normalizedCode := normalizeEnrollmentErrorCode(code)
	if normalizedCode == nil {
		normalizedCode = stringPointer("attendance_enrollment_failed")
	}
	cleanupRequired := session.Operation == attendanceSessionOperationEnroll && (slot != nil || session.StartedAt != nil)
	updates := map[string]any{
		"status":           attendanceSessionStatusFailed,
		"is_active":        false,
		"completed_at":     now,
		"error_code":       *normalizedCode,
		"error_message":    normalizeEnrollmentErrorMessage(message),
		"cleanup_required": cleanupRequired,
		"updated_at":       now,
	}
	if slot != nil {
		updates["fingerprint_slot"] = *slot
	}
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceEnrollmentSession{}).
		Where("id = ? AND is_active = true", session.ID).Updates(updates).Error; err != nil {
		s.Log.Error("attendance enrollment failure update failed", "session_id", session.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to record attendance enrollment failure")
		return
	}
	var updated models.AttendanceEnrollmentSession
	if err := s.DB.WithContext(c.Request.Context()).First(&updated, session.ID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load attendance enrollment")
		return
	}
	s.writeDeviceAudit(c, "attendance_enrollment_failed", "AttendanceEnrollmentSession", &updated.ID, gin.H{
		"device_id":        device.ID,
		"session_uid":      updated.SessionUID,
		"error_code":       updated.ErrorCode,
		"error_message":    updated.ErrorMessage,
		"fingerprint_slot": updated.FingerprintSlot,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Attendance enrollment failure recorded",
		"enrollment": serializeAttendanceEnrollmentSession(&updated, nil, device),
	})
}

func (s *Server) failAttendanceEnrollmentSession(c *gin.Context) {
	device, session, ok := s.loadAttendanceDeviceSession(c)
	if !ok {
		return
	}
	var request struct {
		FingerprintSlot *int   `json:"fingerprint_slot"`
		ErrorCode       string `json:"error_code"`
		ErrorMessage    string `json:"error_message"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if request.FingerprintSlot != nil && !validateEnrollmentSlot(*request.FingerprintSlot) {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint slot must be between 1 and 127")
		return
	}
	now := time.Now().UTC()
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := expireAttendanceSession(s.DB.WithContext(c.Request.Context()), session, now); err != nil {
			s.Log.Error("attendance session expiry failed", "session_id", session.ID, "error", err)
		}
		fail(c, http.StatusConflict, "Attendance enrollment session expired")
		return
	}
	if !session.IsActive {
		fail(c, http.StatusConflict, "Attendance enrollment is no longer active")
		return
	}
	s.applyAttendanceEnrollmentFailure(c, device, session, request.FingerprintSlot, request.ErrorCode, request.ErrorMessage)
}

func (s *Server) completeAttendanceEnrollmentSession(c *gin.Context) {
	device, session, ok := s.loadAttendanceDeviceSession(c)
	if !ok {
		return
	}
	var request struct {
		FingerprintSlot int    `json:"fingerprint_slot"`
		DeviceUserID    string `json:"device_user_id"`
	}
	if !bindJSON(c, &request) {
		return
	}
	ctx := c.Request.Context()
	now := time.Now().UTC()
	if session.Status == attendanceSessionStatusCompleted {
		var member models.Member
		var mapping *models.BiometricEnrollment
		_ = s.DB.WithContext(ctx).First(&member, session.MemberID).Error
		if session.Operation == attendanceSessionOperationEnroll {
			var loaded models.BiometricEnrollment
			if err := s.DB.WithContext(ctx).Where("member_id = ? AND device_registration_id = ? AND reader_key = ?", session.MemberID, device.ID, attendanceReaderKey).First(&loaded).Error; err == nil {
				mapping = &loaded
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "Attendance enrollment is already completed",
			"enrollment": serializeAttendanceEnrollmentSession(session, &member, device),
			"mapping":    serializeAttendanceMapping(mapping, &member),
		})
		return
	}
	if !session.IsActive || session.Status == attendanceSessionStatusFailed ||
		session.Status == attendanceSessionStatusCancelled || session.Status == attendanceSessionStatusExpired {
		fail(c, http.StatusConflict, "Attendance enrollment is no longer active")
		return
	}
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := expireAttendanceSession(s.DB.WithContext(ctx), session, now); err != nil {
			s.Log.Error("attendance session expiry failed", "session_id", session.ID, "error", err)
		}
		fail(c, http.StatusConflict, "Attendance enrollment session expired")
		return
	}

	slot := request.FingerprintSlot
	if slot == 0 && session.FingerprintSlot != nil {
		slot = *session.FingerprintSlot
	}
	if slot != 0 && !validateEnrollmentSlot(slot) {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint slot must be between 1 and 127")
		return
	}
	if session.Operation == attendanceSessionOperationEnroll && slot == 0 {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint slot is required")
		return
	}

	var member models.Member
	if err := s.DB.WithContext(ctx).First(&member, session.MemberID).Error; err != nil {
		fail(c, http.StatusNotFound, "Member not found")
		return
	}

	var mapping *models.BiometricEnrollment
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockAttendanceMember(tx, member.ID); err != nil {
			return err
		}
		if session.Operation == attendanceSessionOperationDelete {
			if err := tx.Where("member_id = ? AND device_registration_id = ? AND reader_key = ?", member.ID, device.ID, attendanceReaderKey).
				Delete(&models.BiometricEnrollment{}).Error; err != nil {
				return err
			}
			mapping = nil
		} else {
			var existing models.BiometricEnrollment
			err := tx.Where("member_id = ? AND device_registration_id = ?", member.ID, device.ID).First(&existing).Error
			if err == nil {
				if existing.ReaderKey == nil || *existing.ReaderKey != attendanceReaderKey {
					return errAttendanceNonAttendanceMapping
				}
				if existing.FingerprintSlot == nil || *existing.FingerprintSlot != slot {
					return errAttendanceMemberSlotConflict
				}
				if request.DeviceUserID != "" {
					if err := tx.Model(&models.BiometricEnrollment{}).Where("id = ?", existing.ID).
						Update("device_user_id", request.DeviceUserID).Error; err != nil {
						return err
					}
					existing.DeviceUserID = request.DeviceUserID
				}
				mapping = &existing
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				var slotOwner models.BiometricEnrollment
				slotErr := tx.Where("device_registration_id = ? AND fingerprint_slot = ?", device.ID, slot).First(&slotOwner).Error
				if slotErr == nil {
					if slotOwner.MemberID != member.ID {
						return errAttendanceSlotTaken
					}
				} else if !errors.Is(slotErr, gorm.ErrRecordNotFound) {
					return slotErr
				}
				deviceUserID := strings.TrimSpace(request.DeviceUserID)
				if deviceUserID == "" {
					deviceUserID = member.LoginID
				}
				readerKey := attendanceReaderKey
				enrollment := models.BiometricEnrollment{
					MemberID:             member.ID,
					DeviceRegistrationID: device.ID,
					ReaderKey:            &readerKey,
					DeviceUserID:         deviceUserID,
					FingerprintSlot:      &slot,
					Status:               "enrolled",
				}
				if err := tx.Create(&enrollment).Error; err != nil {
					return err
				}
				mapping = &enrollment
			} else {
				return err
			}
		}

		updates := map[string]any{
			"status":           attendanceSessionStatusCompleted,
			"is_active":        false,
			"completed_at":     now,
			"cleanup_required": false,
			"updated_at":       now,
		}
		if session.Operation == attendanceSessionOperationEnroll {
			updates["fingerprint_slot"] = slot
			if mapping != nil {
				updates["device_user_id"] = mapping.DeviceUserID
			}
		}
		if err := tx.Model(&models.AttendanceEnrollmentSession{}).Where("id = ? AND is_active = true", session.ID).Updates(updates).Error; err != nil {
			return err
		}
		session.Status = attendanceSessionStatusCompleted
		session.IsActive = false
		session.CompletedAt = &now
		session.CleanupRequired = false
		if session.Operation == attendanceSessionOperationEnroll {
			session.FingerprintSlot = &slot
			if mapping != nil {
				session.DeviceUserID = &mapping.DeviceUserID
			}
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errAttendanceNonAttendanceMapping),
			errors.Is(err, errAttendanceMemberSlotConflict),
			errors.Is(err, errAttendanceSlotTaken):
			fail(c, http.StatusConflict, err.Error())
		default:
			s.Log.Error("attendance enrollment completion failed", "session_id", session.ID, "error", err)
			fail(c, http.StatusInternalServerError, "Unable to complete attendance enrollment")
		}
		return
	}

	action := "attendance_fingerprint_enrolled"
	if session.Operation == attendanceSessionOperationDelete {
		action = "attendance_fingerprint_removed"
	}
	s.writeDeviceAudit(c, action, "Member", &member.ID, gin.H{
		"device_id":        device.ID,
		"session_uid":      session.SessionUID,
		"fingerprint_slot": session.FingerprintSlot,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Attendance enrollment completed",
		"enrollment": serializeAttendanceEnrollmentSession(session, &member, device),
		"mapping":    serializeAttendanceMapping(mapping, &member),
	})
}

func (s *Server) acknowledgeAttendanceEnrollmentCleanup(c *gin.Context) {
	device, session, ok := s.loadAttendanceDeviceSession(c)
	if !ok {
		return
	}
	var request struct {
		Success      bool   `json:"success"`
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	if !bindJSON(c, &request) {
		return
	}
	now := time.Now().UTC()
	if !session.CleanupRequired || session.CleanupCompletedAt != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "No attendance cleanup is pending",
			"enrollment": serializeAttendanceEnrollmentSession(session, nil, device),
		})
		return
	}
	if !request.Success {
		updates := map[string]any{
			"error_code":    normalizeEnrollmentErrorCode(request.ErrorCode),
			"error_message": normalizeEnrollmentErrorMessage(request.ErrorMessage),
			"updated_at":    now,
		}
		if err := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceEnrollmentSession{}).
			Where("id = ?", session.ID).Updates(updates).Error; err != nil {
			s.Log.Error("attendance cleanup failure update failed", "session_id", session.ID, "error", err)
			fail(c, http.StatusInternalServerError, "Unable to update attendance cleanup")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "Attendance cleanup failure recorded",
			"enrollment": serializeAttendanceEnrollmentSession(session, nil, device),
		})
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceEnrollmentSession{}).
		Where("id = ?", session.ID).Updates(map[string]any{
		"cleanup_completed_at": now,
		"cleanup_required":     true,
		"updated_at":           now,
	}).Error; err != nil {
		s.Log.Error("attendance cleanup acknowledgement failed", "session_id", session.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to acknowledge attendance cleanup")
		return
	}
	session.CleanupCompletedAt = &now
	s.writeDeviceAudit(c, "attendance_enrollment_cleanup_completed", "AttendanceEnrollmentSession", &session.ID, gin.H{
		"device_id":   device.ID,
		"session_uid": session.SessionUID,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Attendance cleanup acknowledged",
		"enrollment": serializeAttendanceEnrollmentSession(session, nil, device),
	})
}

func (s *Server) attendanceTerminalHeartbeat(c *gin.Context) {
	device, ok := attendanceTerminalFromContext(c)
	if !ok {
		return
	}
	if !s.allowAttendanceDeviceRequest(c, device, "heartbeat", attendanceHeartbeatRate, time.Minute) {
		return
	}
	var request struct {
		FirmwareVersion string `json:"firmware_version"`
		ReaderModel     string `json:"reader_model"`
		ReaderFirmware  string `json:"reader_firmware"`
		ReaderConnected bool   `json:"reader_connected"`
		ReaderReadiness string `json:"reader_readiness"`
		ReaderKey       string `json:"reader_key"`
		ReaderLabel     string `json:"reader_label"`
		ReaderUART      int    `json:"reader_uart"`
		QueueDepth      int    `json:"queue_depth"`
		NetworkStatus   string `json:"network_status"`
		LastSyncAt      string `json:"last_sync_at"`
		LastCaptureAt   string `json:"last_capture_at"`
	}
	if !bindJSON(c, &request) {
		return
	}
	now := time.Now().UTC()
	readiness := normalizeDeviceReaderReadiness(request.ReaderReadiness)
	readerKey := normalizeEnrollmentReaderKey(request.ReaderKey)
	if readerKey == "" {
		readerKey = attendanceReaderKey
	}
	// Attendance terminals own a single attendance reader. They must not write
	// rows that look like the door controller's login/logout readers.
	if readerKey != attendanceReaderKey {
		fail(c, http.StatusUnprocessableEntity, "Attendance terminals may only report the attendance reader")
		return
	}
	queueDepth := request.QueueDepth
	if queueDepth < 0 {
		queueDepth = 0
	}
	lastSync := parseAttendanceTimestamp(request.LastSyncAt)
	lastCapture := parseAttendanceTimestamp(request.LastCaptureAt)
	status := models.AttendanceTerminalStatus{
		DeviceRegistrationID: device.ID,
		FirmwareVersion:      intakeText(request.FirmwareVersion, 64),
		ReaderModel:          intakeText(request.ReaderModel, 64),
		ReaderFirmware:       intakeText(request.ReaderFirmware, 64),
		ReaderConnected:      request.ReaderConnected,
		ReaderReadiness:      readiness,
		QueueDepth:           queueDepth,
		NetworkStatus:        intakeText(request.NetworkStatus, 32),
		LastSyncAt:           lastSync,
		LastCaptureAt:        lastCapture,
		ReportedAt:           &now,
		UpdatedAt:            now,
	}
	readerLabel := intakeText(request.ReaderLabel, 64)
	if readerLabel == "" {
		readerLabel = enrollmentReaderLabel(readerKey)
	}
	readerUART := request.ReaderUART
	if readerUART == 0 {
		_, defaultUART := enrollmentReaderDefaults(readerKey)
		readerUART = defaultUART
	}
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "device_registration_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"firmware_version": status.FirmwareVersion,
				"reader_model":     status.ReaderModel,
				"reader_firmware":  status.ReaderFirmware,
				"reader_connected": status.ReaderConnected,
				"reader_readiness": status.ReaderReadiness,
				"queue_depth":      status.QueueDepth,
				"network_status":   status.NetworkStatus,
				"last_sync_at":     status.LastSyncAt,
				"last_capture_at":  status.LastCaptureAt,
				"reported_at":      now,
				"updated_at":       now,
			}),
		}).Create(&status).Error; err != nil {
			return err
		}
		reader := models.DeviceReader{
			DeviceRegistrationID: device.ID,
			ReaderKey:            readerKey,
			Label:                readerLabel,
			UART:                 readerUART,
			Connected:            request.ReaderConnected,
			Readiness:            readiness,
			LastSeenAt:           &now,
			UpdatedAt:            now,
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "device_registration_id"}, {Name: "reader_key"}},
			DoUpdates: clause.Assignments(map[string]any{
				"label":        reader.Label,
				"uart":         reader.UART,
				"connected":    reader.Connected,
				"readiness":    reader.Readiness,
				"last_seen_at": now,
				"updated_at":   now,
			}),
		}).Create(&reader).Error
	})
	if err != nil {
		s.Log.Error("attendance terminal heartbeat failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to record terminal heartbeat")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"server_time": now,
		"terminal":    serializeAttendanceTerminalStatus(&status, device, now),
	})
}

// intakeText bounds device-supplied strings so a misbehaving terminal cannot
// force an oversized write or a database error.
func intakeText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return value
}

func parseAttendanceTimestamp(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}
