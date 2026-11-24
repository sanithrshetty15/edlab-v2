package api

import (
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

var (
	errAttendanceSessionActive         = errors.New("another attendance enrollment session is already active for this terminal")
	errAttendanceMemberAlreadyEnrolled = errors.New("member already has a fingerprint mapping on this terminal")
	errAttendanceMappingNotFound       = errors.New("attendance fingerprint mapping not found")
	errAttendanceMappingSlotMissing    = errors.New("attendance fingerprint mapping has no fingerprint slot")
	errAttendanceNoFreeSlot            = errors.New("no free fingerprint slot is available")
	attendanceOutcomeValues            = map[string]bool{
		attendanceOutcomeEntry:          true,
		attendanceOutcomeExit:           true,
		attendanceOutcomeDuplicate:      true,
		attendanceOutcomeNoMatch:        true,
		attendanceOutcomeUnknownSlot:    true,
		attendanceOutcomeMemberInactive: true,
		attendanceOutcomeUserDisabled:   true,
	}
)

func attendanceDeviceLockKey(deviceID int64) int64 {
	const namespace int64 = 0x41545444 // ATTD
	return (namespace << 32) | (deviceID & 0xffffffff)
}

func nextAttendanceSlot(tx *gorm.DB, deviceID int64) (int, error) {
	var slots []int
	if err := tx.Model(&models.BiometricEnrollment{}).
		Where("device_registration_id = ? AND fingerprint_slot IS NOT NULL", deviceID).
		Pluck("fingerprint_slot", &slots).Error; err != nil {
		return 0, err
	}
	used := make(map[int]bool, len(slots))
	for _, slot := range slots {
		used[slot] = true
	}
	for slot := attendanceSlotMin; slot <= attendanceSlotMax; slot++ {
		if !used[slot] {
			return slot, nil
		}
	}
	return 0, errAttendanceNoFreeSlot
}

func serializeAttendanceMapping(mapping *models.BiometricEnrollment, member *models.Member) gin.H {
	if mapping == nil {
		return nil
	}
	data := gin.H{
		"id":                     mapping.ID,
		"member_id":              mapping.MemberID,
		"device_registration_id": mapping.DeviceRegistrationID,
		"reader_key":             mapping.ReaderKey,
		"device_user_id":         mapping.DeviceUserID,
		"fingerprint_slot":       mapping.FingerprintSlot,
		"status":                 mapping.Status,
		"created_at":             mapping.CreatedAt,
		"updated_at":             mapping.UpdatedAt,
	}
	if member != nil {
		data["member"] = gin.H{
			"id":       member.ID,
			"name":     member.Name,
			"login_id": member.LoginID,
			"usn":      member.USN,
			"status":   member.Status,
		}
	}
	return data
}

func serializeAttendanceTerminalStatus(status *models.AttendanceTerminalStatus, device *models.DeviceRegistration, now time.Time) gin.H {
	if status == nil {
		return nil
	}
	online := false
	if device != nil {
		online = deviceIsOnline(device.LastSeenAt, now)
	}
	stale := status.ReportedAt == nil || now.Sub(status.ReportedAt.UTC()) > deviceHeartbeatStaleAfter
	return gin.H{
		"firmware_version": status.FirmwareVersion,
		"reader_model":     status.ReaderModel,
		"reader_firmware":  status.ReaderFirmware,
		"reader_connected": status.ReaderConnected,
		"reader_readiness": status.ReaderReadiness,
		"queue_depth":      status.QueueDepth,
		"network_status":   status.NetworkStatus,
		"last_sync_at":     status.LastSyncAt,
		"last_capture_at":  status.LastCaptureAt,
		"reported_at":      status.ReportedAt,
		"updated_at":       status.UpdatedAt,
		"online":           online,
		"stale":            stale,
	}
}

func (s *Server) loadAttendanceTerminal(c *gin.Context) (*models.DeviceRegistration, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusNotFound, "Attendance terminal not found")
		return nil, false
	}
	var device models.DeviceRegistration
	if err := s.DB.WithContext(c.Request.Context()).First(&device, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Attendance terminal not found")
		return nil, false
	}
	if device.DeviceType != attendanceDeviceType {
		fail(c, http.StatusNotFound, "Attendance terminal not found")
		return nil, false
	}
	return &device, true
}

func attendanceQueryInt64(c *gin.Context, key string) (int64, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return value, nil
}

func (s *Server) listAttendanceLive(c *gin.Context) {
	ctx := c.Request.Context()
	limit := 30
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			fail(c, http.StatusUnprocessableEntity, "limit must be a positive integer")
			return
		}
		limit = value
	}
	if limit > 100 {
		limit = 100
	}
	afterID, err := attendanceQueryInt64(c, "after_id")
	if err != nil {
		fail(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	terminalID, err := attendanceQueryInt64(c, "terminal_id")
	if err != nil {
		fail(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	outcome := strings.ToLower(strings.TrimSpace(c.Query("outcome")))
	if outcome != "" && !attendanceOutcomeValues[outcome] {
		fail(c, http.StatusUnprocessableEntity, "Unknown attendance outcome")
		return
	}

	query := s.DB.WithContext(ctx).
		Preload("DeviceRegistration").
		Where("event_type = ?", attendanceScanEventType)
	if afterID > 0 {
		query = query.Where("id > ?", afterID).Order("id ASC")
	} else {
		query = query.Order("id DESC")
	}
	if terminalID > 0 {
		query = query.Where("device_registration_id = ?", terminalID)
	}
	if outcome != "" {
		query = query.Where("payload->>'outcome' = ?", outcome)
	}
	var events []models.DeviceEvent
	if err := query.Limit(limit).Find(&events).Error; err != nil {
		s.Log.Error("attendance live feed failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load live attendance")
		return
	}
	if afterID == 0 {
		for left, right := 0, len(events)-1; left < right; left, right = left+1, right-1 {
			events[left], events[right] = events[right], events[left]
		}
	}

	identifiers := make([]string, 0, len(events))
	seenMember := make(map[string]bool, len(events))
	for i := range events {
		if events[i].MemberIdentifier == nil {
			continue
		}
		identifier := strings.TrimSpace(*events[i].MemberIdentifier)
		if identifier != "" && !seenMember[identifier] {
			seenMember[identifier] = true
			identifiers = append(identifiers, identifier)
		}
	}
	membersByLogin := make(map[string]models.Member, len(identifiers))
	if len(identifiers) > 0 {
		var members []models.Member
		if err := s.DB.WithContext(ctx).Where("login_id IN ?", identifiers).Find(&members).Error; err != nil {
			s.Log.Error("attendance live member lookup failed", "error", err)
			fail(c, http.StatusInternalServerError, "Unable to load live attendance")
			return
		}
		for i := range members {
			membersByLogin[members[i].LoginID] = members[i]
		}
	}

	items := make([]gin.H, 0, len(events))
	latestID := afterID
	for i := range events {
		event := events[i]
		if event.ID > latestID {
			latestID = event.ID
		}
		payload := attendancePayloadFromEvent(&event)
		var memberData any
		if event.MemberIdentifier != nil {
			if member, ok := membersByLogin[strings.TrimSpace(*event.MemberIdentifier)]; ok {
				memberData = gin.H{
					"id":            member.ID,
					"name":          member.Name,
					"login_id":      member.LoginID,
					"usn":           member.USN,
					"status":        member.Status,
					"academic_year": member.AcademicYear,
					"branch":        member.Branch,
				}
			}
		}
		terminal := gin.H{"id": event.DeviceRegistrationID, "device_uid": "", "name": ""}
		if event.DeviceRegistration.ID > 0 {
			terminal = serializeDevice(&event.DeviceRegistration)
		}
		items = append(items, gin.H{
			"id":                event.ID,
			"event_type":        event.EventType,
			"outcome":           payload.Outcome,
			"reason":            payload.Reason,
			"occurred_at":       event.EventTimestamp,
			"received_at":       event.CreatedAt,
			"synced_late":       payload.SyncedLate,
			"time_source":       payload.TimeSource,
			"fingerprint_slot":  payload.FingerprintSlot,
			"recorded":          attendanceOutcomeRecorded(payload.Outcome),
			"attendance_log_id": payload.AttendanceLogID,
			"member_identifier": event.MemberIdentifier,
			"member":            memberData,
			"terminal":          terminal,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                  true,
		"events":                   items,
		"latest_id":                latestID,
		"server_time":              time.Now().UTC(),
		"online_threshold_seconds": int(deviceHeartbeatStaleAfter.Seconds()),
	})
}

func (s *Server) listAttendanceTerminals(c *gin.Context) {
	ctx := c.Request.Context()
	var devices []models.DeviceRegistration
	if err := s.DB.WithContext(ctx).
		Where("device_type = ?", attendanceDeviceType).
		Order("name ASC, id ASC").Find(&devices).Error; err != nil {
		s.Log.Error("attendance terminal list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance terminals")
		return
	}
	now := time.Now().UTC()
	items := make([]gin.H, 0, len(devices))
	if len(devices) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":                  true,
			"terminals":                items,
			"server_time":              now,
			"online_threshold_seconds": int(deviceHeartbeatStaleAfter.Seconds()),
		})
		return
	}
	deviceIDs := make([]int64, 0, len(devices))
	for i := range devices {
		deviceIDs = append(deviceIDs, devices[i].ID)
	}

	var statuses []models.AttendanceTerminalStatus
	if err := s.DB.WithContext(ctx).Where("device_registration_id IN ?", deviceIDs).Find(&statuses).Error; err != nil {
		s.Log.Error("attendance terminal status list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance terminals")
		return
	}
	statusByDevice := make(map[int64]*models.AttendanceTerminalStatus, len(statuses))
	for i := range statuses {
		statusByDevice[statuses[i].DeviceRegistrationID] = &statuses[i]
	}

	var readers []models.DeviceReader
	if err := s.DB.WithContext(ctx).
		Where("device_registration_id IN ? AND reader_key = ?", deviceIDs, attendanceReaderKey).
		Find(&readers).Error; err != nil {
		s.Log.Error("attendance terminal reader list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance terminals")
		return
	}
	readerByDevice := make(map[int64]models.DeviceReader, len(readers))
	for i := range readers {
		readerByDevice[readers[i].DeviceRegistrationID] = readers[i]
	}

	type mappingCount struct {
		DeviceRegistrationID int64
		Count                int64
	}
	var counts []mappingCount
	if err := s.DB.WithContext(ctx).Model(&models.BiometricEnrollment{}).
		Select("device_registration_id, count(*) AS count").
		Where("device_registration_id IN ? AND reader_key = ?", deviceIDs, attendanceReaderKey).
		Group("device_registration_id").Scan(&counts).Error; err != nil {
		s.Log.Error("attendance mapping count failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance terminals")
		return
	}
	countByDevice := make(map[int64]int64, len(counts))
	for _, count := range counts {
		countByDevice[count.DeviceRegistrationID] = count.Count
	}

	var sessions []models.AttendanceEnrollmentSession
	if err := s.DB.WithContext(ctx).
		Where("device_registration_id IN ? AND (is_active = true OR (cleanup_required = true AND cleanup_completed_at IS NULL))", deviceIDs).
		Order("id ASC").Find(&sessions).Error; err != nil {
		s.Log.Error("attendance terminal session list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance terminals")
		return
	}
	memberIDs := make([]int64, 0, len(sessions))
	seenMembers := make(map[int64]bool, len(sessions))
	for i := range sessions {
		if !seenMembers[sessions[i].MemberID] {
			seenMembers[sessions[i].MemberID] = true
			memberIDs = append(memberIDs, sessions[i].MemberID)
		}
	}
	membersByID := make(map[int64]models.Member, len(memberIDs))
	if len(memberIDs) > 0 {
		var members []models.Member
		if err := s.DB.WithContext(ctx).Find(&members, memberIDs).Error; err != nil {
			s.Log.Error("attendance terminal member lookup failed", "error", err)
			fail(c, http.StatusInternalServerError, "Unable to load attendance terminals")
			return
		}
		for i := range members {
			membersByID[members[i].ID] = members[i]
		}
	}
	activeByDevice := make(map[int64]gin.H, len(sessions))
	cleanupByDevice := make(map[int64]gin.H, len(sessions))
	for i := range sessions {
		session := sessions[i]
		var member *models.Member
		if loaded, ok := membersByID[session.MemberID]; ok {
			member = &loaded
		}
		serialized := serializeAttendanceEnrollmentSession(&session, member, nil)
		if session.IsActive {
			if _, exists := activeByDevice[session.DeviceRegistrationID]; !exists {
				activeByDevice[session.DeviceRegistrationID] = serialized
			}
		}
		if session.CleanupRequired && session.CleanupCompletedAt == nil {
			if _, exists := cleanupByDevice[session.DeviceRegistrationID]; !exists {
				cleanupByDevice[session.DeviceRegistrationID] = serialized
			}
		}
	}

	for i := range devices {
		item := serializeDevice(&devices[i])
		used := countByDevice[devices[i].ID]
		item["mapping_count"] = used
		item["slot_capacity"] = attendanceSlotCapacity
		item["slots_used"] = used
		item["slots_available"] = attendanceSlotCapacity - used
		if status, ok := statusByDevice[devices[i].ID]; ok {
			item["terminal_status"] = serializeAttendanceTerminalStatus(status, &devices[i], now)
		} else {
			item["terminal_status"] = nil
		}
		if reader, ok := readerByDevice[devices[i].ID]; ok {
			available := reader.Connected && enrollmentReaderIsFresh(reader.LastSeenAt, now) &&
				enrollmentReaderCanStart(reader.Readiness)
			item["reader"] = serializeEnrollmentReader(reader, available)
		} else {
			item["reader"] = nil
		}
		if session, ok := activeByDevice[devices[i].ID]; ok {
			item["active_session"] = session
		} else {
			item["active_session"] = nil
		}
		if session, ok := cleanupByDevice[devices[i].ID]; ok {
			item["cleanup_session"] = session
		} else {
			item["cleanup_session"] = nil
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                  true,
		"terminals":                items,
		"server_time":              now,
		"online_threshold_seconds": int(deviceHeartbeatStaleAfter.Seconds()),
	})
}

func (s *Server) listAttendanceTerminalMappings(c *gin.Context) {
	device, ok := s.loadAttendanceTerminal(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	var mappings []models.BiometricEnrollment
	if err := s.DB.WithContext(ctx).
		Where("device_registration_id = ? AND reader_key = ?", device.ID, attendanceReaderKey).
		Order("fingerprint_slot ASC NULLS LAST, id ASC").Find(&mappings).Error; err != nil {
		s.Log.Error("attendance mapping list failed", "terminal_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance mappings")
		return
	}
	memberIDs := make([]int64, 0, len(mappings))
	for i := range mappings {
		memberIDs = append(memberIDs, mappings[i].MemberID)
	}
	membersByID := make(map[int64]models.Member, len(memberIDs))
	if len(memberIDs) > 0 {
		var members []models.Member
		if err := s.DB.WithContext(ctx).Find(&members, memberIDs).Error; err != nil {
			s.Log.Error("attendance mapping member lookup failed", "terminal_id", device.ID, "error", err)
			fail(c, http.StatusInternalServerError, "Unable to load attendance mappings")
			return
		}
		for i := range members {
			membersByID[members[i].ID] = members[i]
		}
	}
	items := make([]gin.H, 0, len(mappings))
	for i := range mappings {
		var member *models.Member
		if loaded, ok := membersByID[mappings[i].MemberID]; ok {
			member = &loaded
		}
		items = append(items, serializeAttendanceMapping(&mappings[i], member))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"terminal":    serializeDevice(device),
		"mappings":    items,
		"server_time": time.Now().UTC(),
	})
}

func (s *Server) createAttendanceEnrollmentSession(c *gin.Context) {
	device, ok := s.loadAttendanceTerminal(c)
	if !ok {
		return
	}
	var request struct {
		MemberID        int64  `json:"member_id"`
		Operation       string `json:"operation"`
		FingerprintSlot *int   `json:"fingerprint_slot"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if request.MemberID <= 0 {
		fail(c, http.StatusUnprocessableEntity, "member_id is required")
		return
	}
	operation := strings.ToLower(strings.TrimSpace(request.Operation))
	if operation == "" {
		operation = attendanceSessionOperationEnroll
	}
	if operation != attendanceSessionOperationEnroll && operation != attendanceSessionOperationDelete {
		fail(c, http.StatusUnprocessableEntity, "operation must be enroll or delete")
		return
	}
	if request.FingerprintSlot != nil && !validateAttendanceSlot(*request.FingerprintSlot) {
		fail(c, http.StatusUnprocessableEntity, "Fingerprint slot must be between 1 and 1000")
		return
	}
	ctx := c.Request.Context()
	var member models.Member
	if err := s.DB.WithContext(ctx).First(&member, request.MemberID).Error; err != nil || member.DeletedAt != nil {
		fail(c, http.StatusNotFound, "Member not found")
		return
	}
	if operation == attendanceSessionOperationEnroll {
		switch member.Status {
		case "active":
			var user models.User
			if err := s.DB.WithContext(ctx).First(&user, member.UserID).Error; err != nil || !user.Active {
				fail(c, http.StatusUnprocessableEntity, "Member account is disabled")
				return
			}
		case memberStatusPendingEnrollment:
		default:
			fail(c, http.StatusUnprocessableEntity, "Member must be active or awaiting fingerprint enrollment")
			return
		}
	}

	now := time.Now().UTC()
	var session models.AttendanceEnrollmentSession
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", attendanceDeviceLockKey(device.ID)).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.AttendanceEnrollmentSession{}).
			Where("device_registration_id = ? AND is_active = true AND expires_at IS NOT NULL AND expires_at <= ?", device.ID, now).
			Updates(map[string]any{
				"status":       attendanceSessionStatusExpired,
				"is_active":    false,
				"completed_at": now,
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}
		var active models.AttendanceEnrollmentSession
		if err := tx.Where("device_registration_id = ? AND is_active = true", device.ID).First(&active).Error; err == nil {
			return errAttendanceSessionActive
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var mapping models.BiometricEnrollment
		mappingErr := tx.Where("member_id = ? AND device_registration_id = ?", member.ID, device.ID).First(&mapping).Error
		var slot int
		switch operation {
		case attendanceSessionOperationDelete:
			if errors.Is(mappingErr, gorm.ErrRecordNotFound) {
				return errAttendanceMappingNotFound
			}
			if mappingErr != nil {
				return mappingErr
			}
			if mapping.ReaderKey == nil || *mapping.ReaderKey != attendanceReaderKey {
				return errAttendanceMappingNotFound
			}
			if mapping.FingerprintSlot == nil {
				return errAttendanceMappingSlotMissing
			}
			slot = *mapping.FingerprintSlot
		default:
			if mappingErr == nil {
				return errAttendanceMemberAlreadyEnrolled
			}
			if !errors.Is(mappingErr, gorm.ErrRecordNotFound) {
				return mappingErr
			}
			if request.FingerprintSlot != nil {
				var count int64
				if err := tx.Model(&models.BiometricEnrollment{}).
					Where("device_registration_id = ? AND fingerprint_slot = ?", device.ID, *request.FingerprintSlot).
					Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return errAttendanceSlotTaken
				}
				slot = *request.FingerprintSlot
			} else {
				allocated, err := nextAttendanceSlot(tx, device.ID)
				if err != nil {
					return err
				}
				slot = allocated
			}
		}

		sessionUID, err := generateEnrollmentSessionUID()
		if err != nil {
			return err
		}
		expiresAt := now.Add(enrollmentSessionTTL)
		readerKey := attendanceReaderKey
		requestedByID := int64(0)
		if user := currentUser(c); user != nil {
			requestedByID = user.ID
		}
		session = models.AttendanceEnrollmentSession{
			SessionUID:           sessionUID,
			MemberID:             member.ID,
			DeviceRegistrationID: device.ID,
			RequestedByID:        requestedByID,
			Operation:            operation,
			Status:               attendanceSessionStatusPending,
			ReaderKey:            &readerKey,
			FingerprintSlot:      &slot,
			IsActive:             true,
			ExpiresAt:            &expiresAt,
		}
		return tx.Create(&session).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, errAttendanceSessionActive),
			errors.Is(err, errAttendanceMemberAlreadyEnrolled),
			errors.Is(err, errAttendanceMappingNotFound),
			errors.Is(err, errAttendanceMappingSlotMissing),
			errors.Is(err, errAttendanceSlotTaken):
			fail(c, http.StatusConflict, err.Error())
		case errors.Is(err, errAttendanceNoFreeSlot):
			fail(c, http.StatusConflict, err.Error())
		default:
			s.Log.Error("attendance enrollment create failed", "terminal_id", device.ID, "error", err)
			fail(c, http.StatusInternalServerError, "Unable to create attendance enrollment session")
		}
		return
	}

	s.audit(c, "attendance_enrollment_requested", "AttendanceEnrollmentSession", &session.ID, gin.H{
		"device_id":        device.ID,
		"session_uid":      session.SessionUID,
		"operation":        session.Operation,
		"member_id":        member.ID,
		"fingerprint_slot": session.FingerprintSlot,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"message":     "Attendance enrollment session created",
		"enrollment":  serializeAttendanceEnrollmentSession(&session, &member, device),
		"server_time": now,
	})
}

func (s *Server) showAttendanceEnrollmentSession(c *gin.Context) {
	device, ok := s.loadAttendanceTerminal(c)
	if !ok {
		return
	}
	uid := strings.TrimSpace(c.Param("sessionId"))
	var session models.AttendanceEnrollmentSession
	if err := s.DB.WithContext(c.Request.Context()).
		Where("session_uid = ? AND device_registration_id = ?", uid, device.ID).First(&session).Error; err != nil {
		fail(c, http.StatusNotFound, "Attendance enrollment session not found")
		return
	}
	now := time.Now().UTC()
	if session.IsActive && session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
		if err := expireAttendanceSession(s.DB.WithContext(c.Request.Context()), &session, now); err != nil {
			s.Log.Error("attendance session expiry failed", "session_id", session.ID, "error", err)
		}
	}
	var member models.Member
	memberErr := s.DB.WithContext(c.Request.Context()).First(&member, session.MemberID).Error
	var memberPtr *models.Member
	if memberErr == nil {
		memberPtr = &member
	}
	var mapping *models.BiometricEnrollment
	if session.FingerprintSlot != nil {
		var loaded models.BiometricEnrollment
		if err := s.DB.WithContext(c.Request.Context()).
			Where("member_id = ? AND device_registration_id = ? AND reader_key = ? AND fingerprint_slot = ?", session.MemberID, device.ID, attendanceReaderKey, *session.FingerprintSlot).
			First(&loaded).Error; err == nil {
			mapping = &loaded
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"enrollment":  serializeAttendanceEnrollmentSession(&session, memberPtr, device),
		"mapping":     serializeAttendanceMapping(mapping, memberPtr),
		"server_time": now,
	})
}

func (s *Server) cancelAttendanceEnrollmentSession(c *gin.Context) {
	device, ok := s.loadAttendanceTerminal(c)
	if !ok {
		return
	}
	uid := strings.TrimSpace(c.Param("sessionId"))
	var session models.AttendanceEnrollmentSession
	if err := s.DB.WithContext(c.Request.Context()).
		Where("session_uid = ? AND device_registration_id = ?", uid, device.ID).First(&session).Error; err != nil {
		fail(c, http.StatusNotFound, "Attendance enrollment session not found")
		return
	}
	now := time.Now().UTC()
	if !session.IsActive {
		fail(c, http.StatusConflict, "Attendance enrollment session is no longer active")
		return
	}
	cleanupRequired := session.Operation == attendanceSessionOperationEnroll && session.StartedAt != nil
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceEnrollmentSession{}).
		Where("id = ? AND is_active = true", session.ID).Updates(map[string]any{
		"status":           attendanceSessionStatusCancelled,
		"is_active":        false,
		"completed_at":     now,
		"cleanup_required": cleanupRequired,
		"updated_at":       now,
	}).Error; err != nil {
		s.Log.Error("attendance session cancel failed", "session_id", session.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to cancel attendance enrollment")
		return
	}
	session.Status = attendanceSessionStatusCancelled
	session.IsActive = false
	session.CompletedAt = &now
	session.CleanupRequired = cleanupRequired
	session.UpdatedAt = now
	s.audit(c, "attendance_enrollment_cancelled", "AttendanceEnrollmentSession", &session.ID, gin.H{
		"device_id":        device.ID,
		"session_uid":      session.SessionUID,
		"operation":        session.Operation,
		"cleanup_required": cleanupRequired,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Attendance enrollment session cancelled",
		"enrollment":  serializeAttendanceEnrollmentSession(&session, nil, device),
		"server_time": now,
	})
}
