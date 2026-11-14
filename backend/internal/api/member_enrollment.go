package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const (
	memberStatusPendingEnrollment = "pending_enrollment"

	enrollmentStatusPendingDetails   = "pending_details"
	enrollmentStatusAwaitingSensor   = "awaiting_sensor"
	enrollmentStatusWaitingFirstScan = "waiting_first_scan"
	enrollmentStatusFirstScan        = "first_scan_captured"
	enrollmentStatusWaitingRemoval   = "waiting_for_removal"
	enrollmentStatusWaitingSecond    = "waiting_second_scan"
	enrollmentStatusSecondScan       = "second_scan_captured"
	enrollmentStatusVerifying        = "verifying"
	enrollmentStatusFinalized        = "finalized"
	enrollmentStatusFailed           = "failed"
	enrollmentStatusCancelled        = "cancelled"
	enrollmentStatusExpired          = "expired"

	enrollmentSessionTTL = 10 * time.Minute
	enrollmentSlotMin    = 1
	enrollmentSlotMax    = 127
)

var (
	errEnrollmentMemberNotPending = errors.New("member is no longer pending enrollment")
	errEnrollmentSlotInUse        = errors.New("fingerprint slot is already assigned")
	errEnrollmentSlotMismatch     = errors.New("fingerprint slot does not match the reported slot")
	errEnrollmentSessionExpired   = errors.New("enrollment session expired")
	errEnrollmentSessionInactive  = errors.New("enrollment session is not active")
	errEnrollmentReaderRequired   = errors.New("enrollment reader was not selected")
)

var enrollmentProgressStatuses = map[string]bool{
	enrollmentStatusWaitingFirstScan: true,
	enrollmentStatusFirstScan:        true,
	enrollmentStatusWaitingRemoval:   true,
	enrollmentStatusWaitingSecond:    true,
	enrollmentStatusSecondScan:       true,
	enrollmentStatusVerifying:        true,
}

func generateEnrollmentSessionUID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func (s *Server) latestEnrollmentSession(ctx context.Context, memberID int64) (*models.MemberEnrollmentSession, error) {
	var session models.MemberEnrollmentSession
	err := s.DB.WithContext(ctx).Where("member_id = ?", memberID).Order("id DESC").First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Server) attachEnrollmentStatuses(ctx context.Context, members []models.Member) {
	ids := make([]int64, 0, len(members))
	for i := range members {
		if members[i].Status == memberStatusPendingEnrollment {
			ids = append(ids, members[i].ID)
		}
	}
	if len(ids) == 0 {
		return
	}

	var sessions []models.MemberEnrollmentSession
	if err := s.DB.WithContext(ctx).
		Where("member_id IN ?", ids).
		Order("id DESC").
		Find(&sessions).Error; err != nil {
		return
	}
	latest := make(map[int64]string, len(sessions))
	for i := range sessions {
		if _, ok := latest[sessions[i].MemberID]; !ok {
			latest[sessions[i].MemberID] = sessions[i].Status
		}
	}
	for i := range members {
		if members[i].Status != memberStatusPendingEnrollment {
			continue
		}
		if status, ok := latest[members[i].ID]; ok {
			members[i].EnrollmentStatus = status
		} else {
			members[i].EnrollmentStatus = enrollmentStatusPendingDetails
		}
	}
}

func serializeEnrollmentSession(session *models.MemberEnrollmentSession, member *models.Member, device *models.DeviceRegistration) gin.H {
	data := gin.H{
		"id":                     session.ID,
		"session_uid":            session.SessionUID,
		"member_id":              session.MemberID,
		"device_registration_id": session.DeviceRegistrationID,
		"reader_key":             session.ReaderKey,
		"status":                 session.Status,
		"fingerprint_slot":       session.FingerprintSlot,
		"device_user_id":         session.DeviceUserID,
		"attempts":               session.Attempts,
		"error_code":             session.ErrorCode,
		"error_message":          session.ErrorMessage,
		"cleanup_required":       session.CleanupRequired,
		"is_active":              session.IsActive,
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

func failEnrollmentState(c *gin.Context, status int, message string, session *models.MemberEnrollmentSession) {
	c.JSON(status, gin.H{
		"success":    false,
		"message":    message,
		"enrollment": serializeEnrollmentSession(session, nil, nil),
	})
}

func (s *Server) resolveEnrollmentDevice(ctx context.Context, requestedID int64) (*models.DeviceRegistration, error) {
	var device models.DeviceRegistration
	query := s.DB.WithContext(ctx).Where("status = ?", deviceStatusActive)
	if requestedID > 0 {
		query = query.Where("id = ?", requestedID)
	}
	if err := query.Order("last_seen_at DESC NULLS LAST, id DESC").First(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

func (s *Server) loadEnrollmentDevice(ctx context.Context, deviceID *int64) *models.DeviceRegistration {
	if deviceID == nil {
		return nil
	}
	var device models.DeviceRegistration
	if err := s.DB.WithContext(ctx).First(&device, *deviceID).Error; err != nil {
		return nil
	}
	return &device
}

func normalizeEnrollmentErrorCode(value string) *string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return nil
	}
	var builder strings.Builder
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '_' || character == '-' {
			builder.WriteRune(character)
		} else {
			builder.WriteByte('_')
		}
		if builder.Len() >= 64 {
			break
		}
	}
	normalized := strings.Trim(builder.String(), "_")
	if normalized == "" {
		return nil
	}
	return &normalized
}

func normalizeEnrollmentErrorMessage(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	runes := []rune(value)
	if len(runes) > 500 {
		value = string(runes[:500])
	}
	return &value
}

func validateEnrollmentSlot(slot int) bool {
	return slot >= enrollmentSlotMin && slot <= enrollmentSlotMax
}

func (s *Server) enqueueEnrollmentExpiry(ctx context.Context, session *models.MemberEnrollmentSession) {
	if s.Jobs == nil || session.ExpiresAt == nil {
		return
	}
	payload, _ := json.Marshal(map[string]int64{"session_id": session.ID})
	if err := s.Jobs.Enqueue(ctx, "biometric.expire_enrollment_session", payload, session.ExpiresAt.Add(time.Second)); err != nil {
		s.Log.Error("failed to enqueue enrollment expiry", "session_id", session.ID, "error", err)
	}
}

func (s *Server) expireEnrollmentSessionRecord(ctx context.Context, session *models.MemberEnrollmentSession, now time.Time) error {
	if session == nil || !session.IsActive {
		return nil
	}
	if session.ExpiresAt == nil || session.ExpiresAt.After(now) {
		return nil
	}
	code := "expired"
	message := "Fingerprint enrollment session expired"
	updates := map[string]any{
		"status":           enrollmentStatusExpired,
		"is_active":        false,
		"completed_at":     now,
		"error_code":       code,
		"error_message":    message,
		"cleanup_required": session.FingerprintSlot != nil,
		"updated_at":       now,
	}
	result := s.DB.WithContext(ctx).Model(&models.MemberEnrollmentSession{}).
		Where("id = ? AND is_active = true", session.ID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		session.Status = enrollmentStatusExpired
		session.IsActive = false
		session.CompletedAt = &now
		session.ErrorCode = &code
		session.ErrorMessage = &message
		session.CleanupRequired = session.FingerprintSlot != nil
	}
	return nil
}

func (s *Server) expireEnrollmentSessionByID(ctx context.Context, sessionID int64) error {
	var session models.MemberEnrollmentSession
	if err := s.DB.WithContext(ctx).First(&session, sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return s.expireEnrollmentSessionRecord(ctx, &session, time.Now().UTC())
}

func (s *Server) expireEnrollmentSessionJob(ctx context.Context, payload json.RawMessage) error {
	var request struct {
		SessionID int64 `json:"session_id"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return fmt.Errorf("decode enrollment expiry job: %w", err)
	}
	if request.SessionID <= 0 {
		return fmt.Errorf("enrollment expiry job requires a session_id")
	}
	return s.expireEnrollmentSessionByID(ctx, request.SessionID)
}

func (s *Server) startMemberFingerprintEnrollment(c *gin.Context) {
	memberID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var request struct {
		DeviceRegistrationID int64  `json:"device_registration_id"`
		ReaderKey            string `json:"reader_key"`
	}
	if !bindOptionalJSON(c, &request) {
		return
	}
	readerKey := normalizeEnrollmentReaderKey(request.ReaderKey)
	if readerKey == "" {
		fail(c, http.StatusUnprocessableEntity, "Select an enrollment reader before starting")
		return
	}
	if !isValidEnrollmentReaderKey(readerKey) {
		fail(c, http.StatusUnprocessableEntity, "Unknown enrollment reader")
		return
	}

	ctx := c.Request.Context()
	var member models.Member
	if err := s.DB.WithContext(ctx).First(&member, memberID).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	now := time.Now().UTC()
	session, err := s.latestEnrollmentSession(ctx, member.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.Log.Error("enrollment session lookup failed", "member_id", member.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to start fingerprint enrollment")
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		session = nil
	}

	if member.Status == "active" {
		if session != nil && session.Status == enrollmentStatusFinalized {
			device := s.loadEnrollmentDevice(ctx, session.DeviceRegistrationID)
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"message":    "Fingerprint enrollment is already complete",
				"member":     serializeManagedMember(&member),
				"enrollment": serializeEnrollmentSession(session, &member, device),
			})
			return
		}
		fail(c, http.StatusConflict, "Member is already active")
		return
	}
	if member.Status != memberStatusPendingEnrollment {
		fail(c, http.StatusUnprocessableEntity, "Only members awaiting fingerprint enrollment can be started")
		return
	}

	if session != nil && session.IsActive {
		if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
			if err := s.expireEnrollmentSessionRecord(ctx, session, now); err != nil {
				s.Log.Error("enrollment session expiry failed", "session_id", session.ID, "error", err)
				fail(c, http.StatusInternalServerError, "Unable to start fingerprint enrollment")
				return
			}
		} else if session.Status != enrollmentStatusPendingDetails {
			if session.ReaderKey == nil || *session.ReaderKey == "" {
				fail(c, http.StatusConflict, "This enrollment has no selected reader. Cancel it and start a new attempt")
				return
			}
			if *session.ReaderKey != readerKey {
				device := s.loadEnrollmentDevice(ctx, session.DeviceRegistrationID)
				c.JSON(http.StatusConflict, gin.H{
					"success":    false,
					"message":    "Another reader is already handling this enrollment session",
					"enrollment": serializeEnrollmentSession(session, &member, device),
				})
				return
			}
			device := s.loadEnrollmentDevice(ctx, session.DeviceRegistrationID)
			member.EnrollmentStatus = session.Status
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"message":    "Fingerprint enrollment is already in progress",
				"member":     serializeManagedMember(&member),
				"enrollment": serializeEnrollmentSession(session, &member, device),
			})
			return
		}
	}

	device, err := s.resolveEnrollmentDevice(ctx, request.DeviceRegistrationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusConflict, "No active biometric device is available for enrollment")
			return
		}
		s.Log.Error("enrollment device lookup failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to start fingerprint enrollment")
		return
	}

	available, err := s.enrollmentReaderAvailable(ctx, device.ID, readerKey, now)
	if err != nil {
		s.Log.Error("enrollment reader availability check failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to start fingerprint enrollment")
		return
	}
	if !available {
		c.JSON(http.StatusConflict, gin.H{
			"success":    false,
			"message":    "The selected enrollment reader is not connected. Select another reader and try again.",
			"code":       "reader_unavailable",
			"reader_key": readerKey,
			"device":     serializeDevice(device),
		})
		return
	}

	expiresAt := now.Add(enrollmentSessionTTL)
	started := false
	if session != nil && session.IsActive && session.Status == enrollmentStatusPendingDetails {
		result := s.DB.WithContext(ctx).Model(&models.MemberEnrollmentSession{}).
			Where("id = ? AND is_active = true", session.ID).
			Updates(map[string]any{
				"device_registration_id": device.ID,
				"reader_key":             readerKey,
				"status":                 enrollmentStatusAwaitingSensor,
				"attempts":               gorm.Expr("attempts + 1"),
				"expires_at":             expiresAt,
				"started_at":             now,
				"error_code":             nil,
				"error_message":          nil,
				"updated_at":             now,
			})
		if result.Error != nil {
			if isDuplicateKeyError(result.Error) {
				fail(c, http.StatusConflict, "Another fingerprint enrollment is already active on this reader. Refresh and try again.")
				return
			}
			s.Log.Error("enrollment session start failed", "session_id", session.ID, "error", result.Error)
			fail(c, http.StatusInternalServerError, "Unable to start fingerprint enrollment")
			return
		}
		if result.RowsAffected == 0 {
			fail(c, http.StatusConflict, "Fingerprint enrollment state changed; please refresh and try again")
			return
		}
		session.DeviceRegistrationID = &device.ID
		session.ReaderKey = &readerKey
		session.Status = enrollmentStatusAwaitingSensor
		session.Attempts++
		session.ExpiresAt = &expiresAt
		session.StartedAt = &now
		session.ErrorCode = nil
		session.ErrorMessage = nil
		started = true
	}

	if !started {
		sessionUID, err := generateEnrollmentSessionUID()
		if err != nil {
			s.Log.Error("enrollment session UID generation failed", "error", err)
			fail(c, http.StatusInternalServerError, "Unable to start fingerprint enrollment")
			return
		}
		user := currentUser(c)
		requestedByID := int64(0)
		if user != nil {
			requestedByID = user.ID
		}
		session = &models.MemberEnrollmentSession{
			SessionUID:           sessionUID,
			MemberID:             member.ID,
			RequestedByID:        requestedByID,
			DeviceRegistrationID: &device.ID,
			ReaderKey:            &readerKey,
			Status:               enrollmentStatusAwaitingSensor,
			Attempts:             1,
			IsActive:             true,
			ExpiresAt:            &expiresAt,
			StartedAt:            &now,
		}
		if err := s.DB.WithContext(ctx).Create(session).Error; err != nil {
			if isDuplicateKeyError(err) {
				fail(c, http.StatusConflict, "Another fingerprint enrollment is already active on this reader. Refresh and try again.")
				return
			}
			s.Log.Error("enrollment session creation failed", "member_id", member.ID, "error", err)
			fail(c, http.StatusInternalServerError, "Unable to start fingerprint enrollment")
			return
		}
	}

	s.enqueueEnrollmentExpiry(ctx, session)
	member.EnrollmentStatus = session.Status
	s.audit(c, "member_fingerprint_enrollment_started", "Member", &member.ID, gin.H{
		"login_id":         member.LoginID,
		"device_id":        device.ID,
		"reader_key":       readerKey,
		"session_uid":      session.SessionUID,
		"session_attempts": session.Attempts,
		"fingerprint_slot": session.FingerprintSlot,
	})

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Fingerprint enrollment started. Place the member's finger on the enrollment sensor.",
		"member":     serializeManagedMember(&member),
		"enrollment": serializeEnrollmentSession(session, &member, device),
	})
}

func (s *Server) memberFingerprintEnrollmentStatus(c *gin.Context) {
	memberID, ok := pathID(c, "id")
	if !ok {
		return
	}
	ctx := c.Request.Context()
	var member models.Member
	if err := s.DB.WithContext(ctx).First(&member, memberID).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	session, err := s.latestEnrollmentSession(ctx, member.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if member.Status == memberStatusPendingEnrollment {
				member.EnrollmentStatus = enrollmentStatusPendingDetails
			}
			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"member":     serializeManagedMember(&member),
				"enrollment": nil,
			})
			return
		}
		fail(c, http.StatusInternalServerError, "Unable to load fingerprint enrollment")
		return
	}

	if session.IsActive {
		now := time.Now().UTC()
		if err := s.expireEnrollmentSessionRecord(ctx, session, now); err != nil {
			s.Log.Error("enrollment session expiry check failed", "session_id", session.ID, "error", err)
		}
	}
	device := s.loadEnrollmentDevice(ctx, session.DeviceRegistrationID)
	if member.Status == memberStatusPendingEnrollment {
		member.EnrollmentStatus = session.Status
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"member":     serializeManagedMember(&member),
		"enrollment": serializeEnrollmentSession(session, &member, device),
	})
}

func (s *Server) cancelMemberFingerprintEnrollment(c *gin.Context) {
	memberID, ok := pathID(c, "id")
	if !ok {
		return
	}
	ctx := c.Request.Context()
	var member models.Member
	if err := s.DB.WithContext(ctx).First(&member, memberID).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	if member.Status != memberStatusPendingEnrollment {
		fail(c, http.StatusUnprocessableEntity, "Only members awaiting fingerprint enrollment can be cancelled")
		return
	}

	now := time.Now().UTC()
	var cancelled *models.MemberEnrollmentSession
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var mappingCount int64
		if err := tx.Model(&models.BiometricEnrollment{}).Where("member_id = ?", member.ID).Count(&mappingCount).Error; err != nil {
			return err
		}
		if mappingCount > 0 {
			return fmt.Errorf("pending member already has a biometric mapping")
		}

		var latest models.MemberEnrollmentSession
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("member_id = ?", member.ID).
			Order("id DESC").
			First(&latest).Error
		if err == nil {
			if latest.IsActive {
				updates := map[string]any{
					"status":           enrollmentStatusCancelled,
					"is_active":        false,
					"completed_at":     now,
					"error_code":       "cancelled",
					"error_message":    "Fingerprint enrollment cancelled by administrator",
					"cleanup_required": latest.FingerprintSlot != nil,
					"updated_at":       now,
				}
				if err := tx.Model(&models.MemberEnrollmentSession{}).
					Where("id = ? AND is_active = true", latest.ID).
					Updates(updates).Error; err != nil {
					return err
				}
				latest.Status = enrollmentStatusCancelled
				latest.IsActive = false
				latest.CompletedAt = &now
				latest.ErrorCode = stringPointer("cancelled")
				latest.ErrorMessage = stringPointer("Fingerprint enrollment cancelled by administrator")
				latest.CleanupRequired = latest.FingerprintSlot != nil
			}
			cancelled = &latest
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Where("user_id = ?", member.UserID).Delete(&models.Session{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Member{}, member.ID).Error; err != nil {
			return err
		}
		return tx.Delete(&models.User{}, member.UserID).Error
	})
	if err != nil {
		s.Log.Error("member registration cancellation failed", "member_id", member.ID, "error", err)
		fail(c, http.StatusUnprocessableEntity, "Unable to cancel registration")
		return
	}

	details := gin.H{
		"login_id": member.LoginID,
		"name":     member.Name,
		"email":    member.Email,
	}
	if cancelled != nil {
		details["session_uid"] = cancelled.SessionUID
		details["cleanup_required"] = cancelled.CleanupRequired
	}
	s.audit(c, "member_registration_cancelled", "Member", &member.ID, details)

	response := gin.H{
		"success": true,
		"message": "Registration cancelled successfully",
	}
	if cancelled != nil {
		response["enrollment"] = serializeEnrollmentSession(cancelled, nil, nil)
	}
	c.JSON(http.StatusOK, response)
}

func stringPointer(value string) *string {
	return &value
}

func (s *Server) cleanupSessionForDevice(ctx context.Context, deviceID int64) (*models.MemberEnrollmentSession, error) {
	var session models.MemberEnrollmentSession
	err := s.DB.WithContext(ctx).
		Where("device_registration_id = ? AND cleanup_required = true AND reader_key IS NOT NULL", deviceID).
		Order("updated_at ASC, id ASC").
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Server) claimEnrollmentSession(c *gin.Context) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return
	}
	ctx := c.Request.Context()

	cleanup, err := s.cleanupSessionForDevice(ctx, device.ID)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"action":      "cleanup",
			"enrollment":  serializeEnrollmentSession(cleanup, nil, device),
			"server_time": time.Now().UTC(),
		})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.Log.Error("enrollment cleanup lookup failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load enrollment session")
		return
	}

	var session models.MemberEnrollmentSession
	err = s.DB.WithContext(ctx).
		Where("device_registration_id = ? AND is_active = true AND reader_key IS NOT NULL", device.ID).
		Order("id ASC").
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"success":     true,
				"action":      "none",
				"enrollment":  nil,
				"server_time": time.Now().UTC(),
			})
			return
		}
		s.Log.Error("active enrollment lookup failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load enrollment session")
		return
	}

	now := time.Now().UTC()
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := s.expireEnrollmentSessionRecord(ctx, &session, now); err != nil {
			s.Log.Error("enrollment session expiry failed", "session_id", session.ID, "error", err)
			fail(c, http.StatusInternalServerError, "Unable to load enrollment session")
			return
		}
		if session.CleanupRequired {
			c.JSON(http.StatusOK, gin.H{
				"success":     true,
				"action":      "cleanup",
				"enrollment":  serializeEnrollmentSession(&session, nil, device),
				"server_time": now,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"action":      "none",
			"enrollment":  nil,
			"server_time": now,
		})
		return
	}

	var member models.Member
	if err := s.DB.WithContext(ctx).First(&member, session.MemberID).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"action":      "enroll",
			"enrollment":  serializeEnrollmentSession(&session, &member, device),
			"server_time": now,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"action":      "enroll",
		"enrollment":  serializeEnrollmentSession(&session, nil, device),
		"server_time": now,
	})
}

func (s *Server) loadDeviceEnrollmentSession(c *gin.Context) (*models.DeviceRegistration, *models.MemberEnrollmentSession, bool) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return nil, nil, false
	}
	uid := strings.TrimSpace(c.Param("uid"))
	if uid == "" {
		fail(c, http.StatusNotFound, "Enrollment session not found")
		return nil, nil, false
	}
	var session models.MemberEnrollmentSession
	if err := s.DB.WithContext(c.Request.Context()).Where("session_uid = ?", uid).First(&session).Error; err != nil {
		fail(c, http.StatusNotFound, "Enrollment session not found")
		return nil, nil, false
	}
	if session.DeviceRegistrationID == nil || *session.DeviceRegistrationID != device.ID {
		fail(c, http.StatusForbidden, "This enrollment session does not belong to this device")
		return nil, nil, false
	}
	return device, &session, true
}

func (s *Server) showEnrollmentSessionForDevice(c *gin.Context) {
	device, session, ok := s.loadDeviceEnrollmentSession(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	now := time.Now().UTC()
	if session.IsActive && session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := s.expireEnrollmentSessionRecord(ctx, session, now); err != nil {
			s.Log.Error("enrollment session expiry failed", "session_id", session.ID, "error", err)
		}
	}

	action := "none"
	switch {
	case session.CleanupRequired:
		action = "cleanup"
	case session.IsActive:
		action = "enroll"
	case session.Status == enrollmentStatusFinalized:
		action = "finalized"
	}
	var member *models.Member
	var loaded models.Member
	if err := s.DB.WithContext(ctx).First(&loaded, session.MemberID).Error; err == nil {
		member = &loaded
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"action":      action,
		"enrollment":  serializeEnrollmentSession(session, member, device),
		"server_time": now,
	})
}

func (s *Server) applyEnrollmentSessionProgress(c *gin.Context, device *models.DeviceRegistration, session *models.MemberEnrollmentSession, status string, slot *int, errorCode, errorMessage string) {
	now := time.Now().UTC()
	updates := map[string]any{
		"status":        status,
		"error_code":    nil,
		"error_message": nil,
		"updated_at":    now,
	}

	if slot != nil {
		if session.FingerprintSlot != nil && *session.FingerprintSlot != *slot {
			failEnrollmentState(c, http.StatusConflict, "Fingerprint slot does not match the slot already reported for this session", session)
			return
		}
		updates["fingerprint_slot"] = *slot
		session.FingerprintSlot = slot
	}

	if status == enrollmentStatusFailed {
		code := normalizeEnrollmentErrorCode(errorCode)
		if code == nil {
			code = stringPointer("enrollment_failed")
		}
		updates["status"] = enrollmentStatusFailed
		updates["is_active"] = false
		updates["completed_at"] = now
		updates["error_code"] = *code
		updates["error_message"] = normalizeEnrollmentErrorMessage(errorMessage)
		updates["cleanup_required"] = session.FingerprintSlot != nil
		session.Status = enrollmentStatusFailed
		session.IsActive = false
		session.CompletedAt = &now
		session.ErrorCode = code
		session.ErrorMessage = normalizeEnrollmentErrorMessage(errorMessage)
		session.CleanupRequired = session.FingerprintSlot != nil
	}

	result := s.DB.WithContext(c.Request.Context()).Model(&models.MemberEnrollmentSession{}).
		Where("id = ? AND is_active = true", session.ID).
		Updates(updates)
	if result.Error != nil {
		s.Log.Error("enrollment progress update failed", "session_id", session.ID, "error", result.Error)
		fail(c, http.StatusInternalServerError, "Unable to update fingerprint enrollment")
		return
	}

	var updated models.MemberEnrollmentSession
	if err := s.DB.WithContext(c.Request.Context()).First(&updated, session.ID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load fingerprint enrollment")
		return
	}
	if result.RowsAffected == 0 && updated.Status != enrollmentStatusFinalized {
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment is no longer active", &updated)
		return
	}

	if updated.Status == enrollmentStatusFailed {
		s.writeDeviceAudit(c, "member_fingerprint_enrollment_failed", "Member", &updated.MemberID, gin.H{
			"device_id":        device.ID,
			"session_uid":      updated.SessionUID,
			"error_code":       updated.ErrorCode,
			"error_message":    updated.ErrorMessage,
			"fingerprint_slot": updated.FingerprintSlot,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Fingerprint enrollment progress recorded",
		"enrollment": serializeEnrollmentSession(&updated, nil, device),
	})
}

func (s *Server) updateEnrollmentSessionProgress(c *gin.Context) {
	device, session, ok := s.loadDeviceEnrollmentSession(c)
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
	if status != enrollmentStatusFailed && !enrollmentProgressStatuses[status] {
		fail(c, http.StatusUnprocessableEntity, "Invalid enrollment status")
		return
	}
	if request.FingerprintSlot != nil && !validateEnrollmentSlot(*request.FingerprintSlot) {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint slot must be between 1 and 127")
		return
	}

	now := time.Now().UTC()
	if session.Status == enrollmentStatusFinalized {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "Fingerprint enrollment is already finalized",
			"enrollment": serializeEnrollmentSession(session, nil, device),
		})
		return
	}
	if session.Status == enrollmentStatusCancelled || session.Status == enrollmentStatusFailed || session.Status == enrollmentStatusExpired {
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment is no longer active", session)
		return
	}
	if !session.IsActive {
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment is no longer active", session)
		return
	}
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := s.expireEnrollmentSessionRecord(c.Request.Context(), session, now); err != nil {
			s.Log.Error("enrollment session expiry failed", "session_id", session.ID, "error", err)
		}
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment session expired", session)
		return
	}

	s.applyEnrollmentSessionProgress(c, device, session, status, request.FingerprintSlot, request.ErrorCode, request.ErrorMessage)
}

func (s *Server) failEnrollmentSession(c *gin.Context) {
	device, session, ok := s.loadDeviceEnrollmentSession(c)
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
	if session.Status == enrollmentStatusFinalized {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "Fingerprint enrollment is already finalized",
			"enrollment": serializeEnrollmentSession(session, nil, device),
		})
		return
	}
	if session.Status == enrollmentStatusCancelled || session.Status == enrollmentStatusFailed || session.Status == enrollmentStatusExpired || !session.IsActive {
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment is no longer active", session)
		return
	}
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := s.expireEnrollmentSessionRecord(c.Request.Context(), session, now); err != nil {
			s.Log.Error("enrollment session expiry failed", "session_id", session.ID, "error", err)
		}
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment session expired", session)
		return
	}
	s.applyEnrollmentSessionProgress(c, device, session, enrollmentStatusFailed, request.FingerprintSlot, request.ErrorCode, request.ErrorMessage)
}

func (s *Server) failEnrollmentSessionRecord(ctx context.Context, sessionID int64, code, message string) error {
	now := time.Now().UTC()
	return s.DB.WithContext(ctx).Model(&models.MemberEnrollmentSession{}).
		Where("id = ? AND is_active = true", sessionID).
		Updates(map[string]any{
			"status":           enrollmentStatusFailed,
			"is_active":        false,
			"completed_at":     now,
			"error_code":       code,
			"error_message":    message,
			"cleanup_required": false,
			"updated_at":       now,
		}).Error
}

func (s *Server) completeEnrollmentSession(c *gin.Context) {
	device, session, ok := s.loadDeviceEnrollmentSession(c)
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
	if !validateEnrollmentSlot(request.FingerprintSlot) {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint slot must be between 1 and 127")
		return
	}
	slot := request.FingerprintSlot
	ctx := c.Request.Context()
	now := time.Now().UTC()

	if session.Status == enrollmentStatusFinalized {
		var member models.Member
		var enrollment models.BiometricEnrollment
		_ = s.DB.WithContext(ctx).First(&member, session.MemberID).Error
		_ = s.DB.WithContext(ctx).Where("member_id = ? AND device_registration_id = ?", session.MemberID, device.ID).First(&enrollment).Error
		s.respondEnrollmentComplete(c, session, &member, &enrollment)
		return
	}
	if session.Status == enrollmentStatusCancelled || session.Status == enrollmentStatusFailed || session.Status == enrollmentStatusExpired {
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment is no longer active", session)
		return
	}
	if !session.IsActive {
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment is no longer active", session)
		return
	}
	if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := s.expireEnrollmentSessionRecord(ctx, session, now); err != nil {
			s.Log.Error("enrollment session expiry failed", "session_id", session.ID, "error", err)
		}
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment session expired", session)
		return
	}
	if session.FingerprintSlot != nil && *session.FingerprintSlot != slot {
		failEnrollmentState(c, http.StatusConflict, "Fingerprint slot does not match the slot reported for this session", session)
		return
	}
	if session.ReaderKey == nil || *session.ReaderKey == "" {
		failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment has no selected reader", session)
		return
	}

	var enrollment models.BiometricEnrollment
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedSession models.MemberEnrollmentSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedSession, session.ID).Error; err != nil {
			return err
		}
		if lockedSession.Status == enrollmentStatusFinalized {
			if err := tx.Where("member_id = ? AND device_registration_id = ?", lockedSession.MemberID, device.ID).First(&enrollment).Error; err != nil {
				return err
			}
			return nil
		}
		if !lockedSession.IsActive {
			return errEnrollmentSessionInactive
		}
		if lockedSession.ExpiresAt != nil && !lockedSession.ExpiresAt.After(now) {
			return errEnrollmentSessionExpired
		}
		if lockedSession.Status == enrollmentStatusCancelled || lockedSession.Status == enrollmentStatusFailed || lockedSession.Status == enrollmentStatusExpired {
			return errEnrollmentSessionInactive
		}
		if lockedSession.FingerprintSlot != nil && *lockedSession.FingerprintSlot != slot {
			return errEnrollmentSlotMismatch
		}
		if lockedSession.ReaderKey == nil || *lockedSession.ReaderKey == "" {
			return errEnrollmentReaderRequired
		}

		var member models.Member
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", lockedSession.MemberID).
			First(&member).Error; err != nil {
			return err
		}
		if member.Status != memberStatusPendingEnrollment {
			return errEnrollmentMemberNotPending
		}

		var slotCount int64
		if err := tx.Model(&models.BiometricEnrollment{}).
			Where("device_registration_id = ? AND fingerprint_slot = ?", device.ID, slot).
			Count(&slotCount).Error; err != nil {
			return err
		}
		if slotCount > 0 {
			return errEnrollmentSlotInUse
		}
		var memberCount int64
		if err := tx.Model(&models.BiometricEnrollment{}).
			Where("member_id = ? AND device_registration_id = ?", member.ID, device.ID).
			Count(&memberCount).Error; err != nil {
			return err
		}
		if memberCount > 0 {
			return errEnrollmentSlotInUse
		}

		deviceUserID := fmt.Sprintf("slot:%d", slot)
		enrollment = models.BiometricEnrollment{
			MemberID:             member.ID,
			DeviceRegistrationID: device.ID,
			ReaderKey:            lockedSession.ReaderKey,
			DeviceUserID:         deviceUserID,
			FingerprintSlot:      &slot,
			Status:               "enrolled",
		}
		if err := tx.Create(&enrollment).Error; err != nil {
			if isDuplicateKeyError(err) {
				return errEnrollmentSlotInUse
			}
			return err
		}
		if err := tx.Model(&models.Member{}).Where("id = ?", member.ID).Update("status", "active").Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", member.UserID).Update("active", true).Error; err != nil {
			return err
		}
		return tx.Model(&models.MemberEnrollmentSession{}).
			Where("id = ?", lockedSession.ID).
			Updates(map[string]any{
				"status":           enrollmentStatusFinalized,
				"is_active":        false,
				"fingerprint_slot": slot,
				"device_user_id":   deviceUserID,
				"completed_at":     now,
				"error_code":       nil,
				"error_message":    nil,
				"cleanup_required": false,
				"updated_at":       now,
			}).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, errEnrollmentSlotInUse):
			_ = s.failEnrollmentSessionRecord(ctx, session.ID, "slot_conflict", "Fingerprint slot is already assigned to another member")
			fail(c, http.StatusConflict, "Fingerprint slot is already assigned to another member")
		case errors.Is(err, errEnrollmentMemberNotPending):
			_ = s.failEnrollmentSessionRecord(ctx, session.ID, "member_not_pending", "Member is no longer awaiting fingerprint enrollment")
			fail(c, http.StatusConflict, "Member is no longer awaiting fingerprint enrollment")
		case errors.Is(err, errEnrollmentSlotMismatch):
			fail(c, http.StatusConflict, "Fingerprint slot does not match the slot reported for this session")
		case errors.Is(err, errEnrollmentReaderRequired):
			fail(c, http.StatusConflict, "Fingerprint enrollment has no selected reader")
		case errors.Is(err, errEnrollmentSessionExpired):
			var refreshed models.MemberEnrollmentSession
			_ = s.DB.WithContext(ctx).First(&refreshed, session.ID).Error
			_ = s.expireEnrollmentSessionRecord(ctx, &refreshed, time.Now().UTC())
			failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment session expired", &refreshed)
		case errors.Is(err, errEnrollmentSessionInactive):
			var refreshed models.MemberEnrollmentSession
			_ = s.DB.WithContext(ctx).First(&refreshed, session.ID).Error
			failEnrollmentState(c, http.StatusConflict, "Fingerprint enrollment is no longer active", &refreshed)
		case errors.Is(err, gorm.ErrRecordNotFound):
			fail(c, http.StatusConflict, "Member is no longer available for fingerprint enrollment")
		default:
			s.Log.Error("enrollment finalization failed", "session_id", session.ID, "error", err)
			fail(c, http.StatusInternalServerError, "Unable to finalize fingerprint enrollment")
		}
		return
	}

	var finalized models.MemberEnrollmentSession
	if err := s.DB.WithContext(ctx).First(&finalized, session.ID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load finalized enrollment")
		return
	}
	var member models.Member
	if err := s.DB.WithContext(ctx).First(&member, finalized.MemberID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load member")
		return
	}
	s.writeDeviceAudit(c, "member_fingerprint_enrollment_completed", "Member", &member.ID, gin.H{
		"device_id":        device.ID,
		"reader_key":       finalized.ReaderKey,
		"session_uid":      finalized.SessionUID,
		"fingerprint_slot": finalized.FingerprintSlot,
		"device_user_id":   finalized.DeviceUserID,
	})
	s.respondEnrollmentComplete(c, &finalized, &member, &enrollment)
}

func (s *Server) respondEnrollmentComplete(c *gin.Context, session *models.MemberEnrollmentSession, member *models.Member, enrollment *models.BiometricEnrollment) {
	payload := gin.H{
		"success":    true,
		"message":    "Fingerprint enrollment complete. Member account activated.",
		"enrollment": serializeEnrollmentSession(session, member, nil),
	}
	if member != nil && member.ID > 0 {
		member.EnrollmentStatus = enrollmentStatusFinalized
		var user models.User
		if err := s.DB.WithContext(c.Request.Context()).First(&user, member.UserID).Error; err == nil {
			payload["member"] = serializeManagedMemberFull(member, &user)
		} else {
			payload["member"] = serializeManagedMember(member)
		}
	}
	if enrollment != nil && enrollment.ID > 0 {
		payload["biometric_enrollment"] = gin.H{
			"id":                     enrollment.ID,
			"member_id":              enrollment.MemberID,
			"device_registration_id": enrollment.DeviceRegistrationID,
			"reader_key":             enrollment.ReaderKey,
			"device_user_id":         enrollment.DeviceUserID,
			"fingerprint_slot":       enrollment.FingerprintSlot,
			"status":                 enrollment.Status,
		}
	}
	c.JSON(http.StatusOK, payload)
}

func (s *Server) acknowledgeEnrollmentCleanup(c *gin.Context) {
	device, session, ok := s.loadDeviceEnrollmentSession(c)
	if !ok {
		return
	}
	var request struct {
		FingerprintSlot *int `json:"fingerprint_slot"`
	}
	if !bindOptionalJSON(c, &request) {
		return
	}
	if request.FingerprintSlot != nil {
		if !validateEnrollmentSlot(*request.FingerprintSlot) {
			fail(c, http.StatusUnprocessableEntity, "Fingerprint slot must be between 1 and 127")
			return
		}
		if session.FingerprintSlot != nil && *session.FingerprintSlot != *request.FingerprintSlot {
			fail(c, http.StatusUnprocessableEntity, "Fingerprint slot does not match the enrollment session")
			return
		}
	}
	if !session.CleanupRequired {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"message":    "Enrollment cleanup is already complete",
			"enrollment": serializeEnrollmentSession(session, nil, device),
		})
		return
	}

	now := time.Now().UTC()
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.MemberEnrollmentSession{}).
		Where("id = ?", session.ID).
		Updates(map[string]any{
			"cleanup_required":     false,
			"cleanup_completed_at": now,
			"updated_at":           now,
		}).Error; err != nil {
		s.Log.Error("enrollment cleanup acknowledgement failed", "session_id", session.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to acknowledge enrollment cleanup")
		return
	}
	session.CleanupRequired = false
	session.CleanupCompletedAt = &now
	s.writeDeviceAudit(c, "member_fingerprint_enrollment_cleanup_completed", "Member", &session.MemberID, gin.H{
		"device_id":        device.ID,
		"session_uid":      session.SessionUID,
		"fingerprint_slot": session.FingerprintSlot,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Enrollment cleanup acknowledged",
		"enrollment": serializeEnrollmentSession(session, nil, device),
	})
}
