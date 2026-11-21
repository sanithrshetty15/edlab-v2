package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const (
	attendanceDeviceType = "attendance_terminal"
	attendanceReaderKey  = "attendance"

	attendanceScanEventType = "attendance_scan"

	attendanceOutcomeEntry          = "entry"
	attendanceOutcomeExit           = "exit"
	attendanceOutcomeDuplicate      = "duplicate"
	attendanceOutcomeNoMatch        = "no_match"
	attendanceOutcomeUnknownSlot    = "unknown_slot"
	attendanceOutcomeMemberInactive = "member_inactive"
	attendanceOutcomeUserDisabled   = "user_disabled"

	attendanceSessionOperationEnroll = "enroll"
	attendanceSessionOperationDelete = "delete"

	attendanceSessionStatusPending    = "pending"
	attendanceSessionStatusInProgress = "in_progress"
	attendanceSessionStatusCompleted  = "completed"
	attendanceSessionStatusFailed     = "failed"
	attendanceSessionStatusCancelled  = "cancelled"
	attendanceSessionStatusExpired    = "expired"

	attendanceDedupeWindow      = 10 * time.Second
	attendanceSyncedLateAfter   = 60 * time.Second
	attendanceClockSkew         = 5 * time.Minute
	attendanceMaxBacklogAge     = 7 * 24 * time.Hour
	attendanceBatchLimit        = 50
	attendanceScanRateLimit     = 120
	attendanceBatchRateLimit    = 30
	attendanceHeartbeatRate     = 120
	attendancePollRateLimit     = 120
	attendanceEnrollmentCommand = 60
)

var (
	attendanceIdempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

	errAttendanceIdempotencyConflict = errors.New("idempotency key belongs to another device")
	errAttendanceInvalidEvent        = errors.New("event is not an attendance scan")
)

type attendanceScanRequest struct {
	IdempotencyKey  string `json:"idempotency_key"`
	FingerprintSlot *int   `json:"fingerprint_slot"`
	Result          string `json:"result"`
	CapturedAt      string `json:"captured_at"`
}

type attendanceScanResult struct {
	Success         bool      `json:"success"`
	Outcome         string    `json:"outcome,omitempty"`
	Recorded        bool      `json:"recorded"`
	Duplicate       bool      `json:"duplicate"`
	EventID         int64     `json:"event_id,omitempty"`
	AttendanceLogID int64     `json:"attendance_log_id,omitempty"`
	Message         string    `json:"message"`
	Retryable       bool      `json:"retryable,omitempty"`
	ServerTime      time.Time `json:"server_time"`
}

type attendanceEventPayload struct {
	Outcome         string `json:"outcome,omitempty"`
	Reason          string `json:"reason,omitempty"`
	FingerprintSlot *int   `json:"fingerprint_slot,omitempty"`
	CapturedAt      string `json:"captured_at,omitempty"`
	ReceivedAt      string `json:"received_at,omitempty"`
	SyncedLate      bool   `json:"synced_late,omitempty"`
	TimeSource      string `json:"time_source,omitempty"`
	RecordedAt      string `json:"recorded_at,omitempty"`
	AttendanceLogID *int64 `json:"attendance_log_id,omitempty"`
}

type attendanceScanTiming struct {
	CapturedAt time.Time
	OfficialAt time.Time
	Source     string
	SyncedLate bool
}

func attendanceTerminalFromContext(c *gin.Context) (*models.DeviceRegistration, bool) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return nil, false
	}
	if device.DeviceType != attendanceDeviceType {
		fail(c, http.StatusForbidden, "This endpoint requires an attendance terminal")
		return nil, false
	}
	return device, true
}

func (s *Server) allowAttendanceDeviceRequest(c *gin.Context, device *models.DeviceRegistration, bucket string, limit int, window time.Duration) bool {
	if s.Limiter == nil {
		return true
	}
	if !s.Limiter.Allow("attendance:"+bucket+":"+device.DeviceUID, limit, window) {
		c.Header("Retry-After", "60")
		fail(c, http.StatusTooManyRequests, "Too many requests. Please try again later.")
		return false
	}
	return true
}

func normalizeAttendanceScanRequest(request *attendanceScanRequest) error {
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.Result = strings.ToLower(strings.TrimSpace(request.Result))
	request.CapturedAt = strings.TrimSpace(request.CapturedAt)

	if !attendanceIdempotencyPattern.MatchString(request.IdempotencyKey) {
		return errors.New("idempotency_key must be 8-128 letters, digits, dots, underscores, colons or hyphens")
	}
	if request.Result == "" {
		if request.FingerprintSlot != nil {
			request.Result = "match"
		} else {
			request.Result = "no_match"
		}
	}
	switch request.Result {
	case "match":
		if request.FingerprintSlot == nil || !validateEnrollmentSlot(*request.FingerprintSlot) {
			return errors.New("fingerprint_slot must be between 1 and 127 for a matched scan")
		}
	case "no_match":
		if request.FingerprintSlot != nil {
			return errors.New("fingerprint_slot must be omitted for an unmatched scan")
		}
	default:
		return errors.New("result must be match or no_match")
	}
	return nil
}

func resolveAttendanceScanTiming(value string, now time.Time) attendanceScanTiming {
	now = now.UTC()
	timing := attendanceScanTiming{
		CapturedAt: now,
		OfficialAt: now,
		Source:     "server",
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return timing
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return timing
	}
	parsed = parsed.UTC()
	if parsed.After(now.Add(attendanceClockSkew)) {
		timing.Source = "server_clock_skew"
		return timing
	}
	if parsed.Before(now.Add(-attendanceMaxBacklogAge)) {
		timing.Source = "server_too_old"
		return timing
	}
	timing.CapturedAt = parsed
	if now.Sub(parsed) > attendanceSyncedLateAfter {
		timing.OfficialAt = parsed
		timing.SyncedLate = true
		timing.Source = "device"
	}
	return timing
}

func attendanceOutcomeMessage(outcome string) string {
	switch outcome {
	case attendanceOutcomeEntry:
		return "Attendance entry recorded"
	case attendanceOutcomeExit:
		return "Attendance exit recorded"
	case attendanceOutcomeDuplicate:
		return "Duplicate scan ignored"
	case attendanceOutcomeNoMatch:
		return "Fingerprint not recognized"
	case attendanceOutcomeUnknownSlot:
		return "Fingerprint is not enrolled on this terminal"
	case attendanceOutcomeMemberInactive:
		return "Member is not active"
	case attendanceOutcomeUserDisabled:
		return "Member account is disabled"
	default:
		return "Scan received"
	}
}

func payloadLogID(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func attendanceOutcomeRecorded(outcome string) bool {
	return outcome == attendanceOutcomeEntry || outcome == attendanceOutcomeExit
}

func attendanceDeviceReference(device *models.DeviceRegistration) *string {
	value := fmt.Sprintf("attendance_terminal:%s", device.DeviceUID)
	return &value
}

func attendanceAdvisoryLockKey(memberID int64) int64 {
	const namespace int64 = 0x41545445 // ATTE
	return (namespace << 32) | (memberID & 0xffffffff)
}

func lockAttendanceMember(tx *gorm.DB, memberID int64) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", attendanceAdvisoryLockKey(memberID)).Error
}

func attendancePayloadFromEvent(event *models.DeviceEvent) attendanceEventPayload {
	var payload attendanceEventPayload
	if len(event.Payload) > 0 {
		_ = json.Unmarshal(event.Payload, &payload)
	}
	return payload
}

func attendanceResultFromEvent(event *models.DeviceEvent, now time.Time) attendanceScanResult {
	payload := attendancePayloadFromEvent(event)
	outcome := payload.Outcome
	if outcome == "" {
		return attendanceScanResult{
			Success:    true,
			Outcome:    "already_processed",
			EventID:    event.ID,
			Message:    "Event was already processed",
			ServerTime: now,
		}
	}
	return attendanceScanResult{
		Success:         true,
		Outcome:         outcome,
		Recorded:        attendanceOutcomeRecorded(outcome),
		Duplicate:       outcome == attendanceOutcomeDuplicate,
		EventID:         event.ID,
		AttendanceLogID: payloadLogID(payload.AttendanceLogID),
		Message:         attendanceOutcomeMessage(outcome),
		ServerTime:      now,
	}
}

func (s *Server) createAttendanceEvent(tx *gorm.DB, device *models.DeviceRegistration, request attendanceScanRequest, timing attendanceScanTiming, memberIdentifier *string, initialOutcome string) (*models.DeviceEvent, bool, error) {
	payload, err := json.Marshal(attendanceEventPayload{
		Outcome:         initialOutcome,
		Reason:          initialOutcome,
		FingerprintSlot: request.FingerprintSlot,
		CapturedAt:      timing.CapturedAt.Format(time.RFC3339Nano),
		ReceivedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		SyncedLate:      timing.SyncedLate,
		TimeSource:      timing.Source,
	})
	if err != nil {
		return nil, false, err
	}
	event := models.DeviceEvent{
		DeviceRegistrationID: device.ID,
		EventType:            attendanceScanEventType,
		IdempotencyKey:       request.IdempotencyKey,
		MemberIdentifier:     memberIdentifier,
		EventTimestamp:       timing.CapturedAt,
		Payload:              payload,
		ProcessingStatus:     "pending",
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(&event)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected > 0 {
		return &event, false, nil
	}

	var existing models.DeviceEvent
	if err := tx.Where("idempotency_key = ?", request.IdempotencyKey).First(&existing).Error; err != nil {
		return nil, false, err
	}
	if existing.DeviceRegistrationID != device.ID {
		return nil, false, errAttendanceIdempotencyConflict
	}
	if existing.EventType != attendanceScanEventType {
		return nil, false, errAttendanceInvalidEvent
	}
	return &existing, true, nil
}

func (s *Server) processAttendanceScan(ctx context.Context, device *models.DeviceRegistration, request attendanceScanRequest) (attendanceScanResult, int, error) {
	now := time.Now().UTC()
	if err := normalizeAttendanceScanRequest(&request); err != nil {
		return attendanceScanResult{Success: false, Message: err.Error(), ServerTime: now}, http.StatusUnprocessableEntity, err
	}
	timing := resolveAttendanceScanTiming(request.CapturedAt, now)

	var result attendanceScanResult
	var auditMember *models.Member
	var auditEvent *models.DeviceEvent

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		initialOutcome := ""
		var member *models.Member
		var memberIdentifier *string

		if request.Result == "no_match" {
			initialOutcome = attendanceOutcomeNoMatch
		} else {
			var enrollment models.BiometricEnrollment
			err := tx.Where(
				"device_registration_id = ? AND fingerprint_slot = ? AND status IN ? AND reader_key = ?",
				device.ID,
				*request.FingerprintSlot,
				[]string{"enrolled", "verified"},
				attendanceReaderKey,
			).First(&enrollment).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				initialOutcome = attendanceOutcomeUnknownSlot
			} else if err != nil {
				return err
			} else {
				var loaded models.Member
				err = tx.Where("id = ? AND deleted_at IS NULL", enrollment.MemberID).First(&loaded).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					initialOutcome = attendanceOutcomeMemberInactive
				} else if err != nil {
					return err
				} else {
					member = &loaded
					memberIdentifier = stringPointer(loaded.LoginID)
					if loaded.Status != "active" {
						initialOutcome = attendanceOutcomeMemberInactive
					} else {
						var user models.User
						userErr := tx.First(&user, loaded.UserID).Error
						if errors.Is(userErr, gorm.ErrRecordNotFound) || (userErr == nil && !user.Active) {
							initialOutcome = attendanceOutcomeUserDisabled
						} else if userErr != nil {
							return userErr
						}
					}
				}
			}
		}

		event, existing, err := s.createAttendanceEvent(tx, device, request, timing, memberIdentifier, initialOutcome)
		if err != nil {
			return err
		}
		if existing {
			result = attendanceResultFromEvent(event, now)
			return nil
		}

		payload := attendanceEventPayload{
			FingerprintSlot: request.FingerprintSlot,
			CapturedAt:      timing.CapturedAt.Format(time.RFC3339Nano),
			ReceivedAt:      now.Format(time.RFC3339Nano),
			SyncedLate:      timing.SyncedLate,
			TimeSource:      timing.Source,
		}
		outcome := initialOutcome
		if outcome == "" && member != nil {
			if err := lockAttendanceMember(tx, member.ID); err != nil {
				return err
			}
			applied, err := s.applyAttendanceRecord(
				tx,
				member,
				timing.OfficialAt,
				timing.OfficialAt,
				attendanceDeviceReference(device),
				attendanceDedupeWindow,
			)
			if err != nil {
				return err
			}
			outcome = applied.Outcome
			payload.RecordedAt = timing.OfficialAt.Format(time.RFC3339Nano)
			payload.AttendanceLogID = &applied.Log.ID
			auditMember = member
			auditEvent = event
		}
		if outcome == "" {
			outcome = attendanceOutcomeUnknownSlot
		}
		payload.Outcome = outcome
		payload.Reason = outcome
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if err := tx.Model(&models.DeviceEvent{}).Where("id = ?", event.ID).Updates(map[string]any{
			"payload":           encoded,
			"processing_status": "completed",
			"processing_error":  nil,
		}).Error; err != nil {
			return err
		}
		event.Payload = encoded
		event.ProcessingStatus = "completed"

		result = attendanceScanResult{
			Success:         true,
			Outcome:         outcome,
			Recorded:        attendanceOutcomeRecorded(outcome),
			Duplicate:       outcome == attendanceOutcomeDuplicate,
			EventID:         event.ID,
			AttendanceLogID: payloadLogID(payload.AttendanceLogID),
			Message:         attendanceOutcomeMessage(outcome),
			ServerTime:      now,
		}
		return nil
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errAttendanceIdempotencyConflict) || errors.Is(err, errAttendanceInvalidEvent) {
			status = http.StatusConflict
		}
		return attendanceScanResult{Success: false, Message: err.Error(), ServerTime: now}, status, err
	}

	if auditMember != nil && auditEvent != nil && result.EventID > 0 {
		action := "attendance_login_biometric"
		if result.Outcome == attendanceOutcomeExit {
			action = "attendance_logout_biometric"
		}
		if result.Recorded && result.AttendanceLogID > 0 {
			s.writeSystemAudit(ctx, action, "AttendanceLog", &result.AttendanceLogID, *auditMember, auditEvent)
		}
	}
	return result, http.StatusOK, nil
}

func (s *Server) submitAttendanceScan(c *gin.Context) {
	device, ok := attendanceTerminalFromContext(c)
	if !ok {
		return
	}
	if !s.allowAttendanceDeviceRequest(c, device, "scan", attendanceScanRateLimit, time.Minute) {
		return
	}
	var request attendanceScanRequest
	if !bindJSON(c, &request) {
		return
	}
	result, status, err := s.processAttendanceScan(c.Request.Context(), device, request)
	if err != nil {
		if status >= http.StatusInternalServerError {
			s.Log.Error("attendance scan failed", "device_id", device.ID, "error", err)
			fail(c, status, "Unable to record attendance")
			return
		}
		fail(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"outcome":           result.Outcome,
		"recorded":          result.Recorded,
		"duplicate":         result.Duplicate,
		"event_id":          result.EventID,
		"attendance_log_id": result.AttendanceLogID,
		"message":           result.Message,
		"server_time":       result.ServerTime,
	})
}

func (s *Server) syncAttendanceScans(c *gin.Context) {
	device, ok := attendanceTerminalFromContext(c)
	if !ok {
		return
	}
	if !s.allowAttendanceDeviceRequest(c, device, "sync", attendanceBatchRateLimit, time.Minute) {
		return
	}
	var request struct {
		Scans []attendanceScanRequest `json:"scans"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if len(request.Scans) == 0 {
		fail(c, http.StatusUnprocessableEntity, "At least one scan is required")
		return
	}
	if len(request.Scans) > attendanceBatchLimit {
		fail(c, http.StatusUnprocessableEntity, fmt.Sprintf("A batch may contain at most %d scans", attendanceBatchLimit))
		return
	}

	results := make([]gin.H, 0, len(request.Scans))
	for _, scan := range request.Scans {
		result, status, err := s.processAttendanceScan(c.Request.Context(), device, scan)
		item := gin.H{"idempotency_key": strings.TrimSpace(scan.IdempotencyKey)}
		if err != nil {
			if status >= http.StatusInternalServerError {
				s.Log.Error("attendance batch scan failed", "device_id", device.ID, "error", err)
			}
			item["success"] = false
			item["message"] = err.Error()
			item["retryable"] = status >= http.StatusInternalServerError
		} else {
			item["success"] = true
			item["outcome"] = result.Outcome
			item["recorded"] = result.Recorded
			item["duplicate"] = result.Duplicate
			item["event_id"] = result.EventID
			item["attendance_log_id"] = result.AttendanceLogID
			item["message"] = result.Message
		}
		results = append(results, item)
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"results":     results,
		"server_time": time.Now().UTC(),
	})
}
