package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const (
	enrollmentReaderLogin      = "login"
	enrollmentReaderLogout     = "logout"
	enrollmentReaderDedicated  = "dedicated"
	enrollmentReaderAttendance = "attendance"

	enrollmentReaderFreshness = 90 * time.Second
)

func enrollmentReaderDefaults(key string) (label string, uart int) {
	switch key {
	case enrollmentReaderLogin:
		return "Login reader", 2
	case enrollmentReaderLogout:
		return "Logout reader", 1
	case enrollmentReaderDedicated:
		return "Dedicated enrollment reader", 0
	case enrollmentReaderAttendance:
		return "Attendance reader", 0
	default:
		return "", 0
	}
}

func isValidEnrollmentReaderKey(key string) bool {
	switch key {
	case enrollmentReaderLogin, enrollmentReaderLogout, enrollmentReaderDedicated, enrollmentReaderAttendance:
		return true
	default:
		return false
	}
}

func normalizeEnrollmentReaderKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func enrollmentReaderLabel(key string) string {
	if label, _ := enrollmentReaderDefaults(key); label != "" {
		return label
	}
	return "Selected reader"
}

func enrollmentReaderIsFresh(lastSeen *time.Time, now time.Time) bool {
	return lastSeen != nil && !lastSeen.IsZero() && now.Sub(lastSeen.UTC()) <= enrollmentReaderFreshness
}

func serializeEnrollmentReader(reader models.DeviceReader, available bool) gin.H {
	readiness := strings.TrimSpace(reader.Readiness)
	if readiness == "" {
		readiness = "not_ready"
	}
	return gin.H{
		"key":          reader.ReaderKey,
		"label":        reader.Label,
		"uart":         reader.UART,
		"connected":    reader.Connected,
		"available":    available,
		"readiness":    readiness,
		"last_seen_at": reader.LastSeenAt,
	}
}

// enrollmentReaderCanStart mirrors the firmware contract: a sensor may only be
// offered for enrollment after it has answered at least one command. A sensor
// that is still initializing or currently failing capture must not start a
// session that is guaranteed to abort.
func enrollmentReaderCanStart(readiness string) bool {
	switch strings.ToLower(strings.TrimSpace(readiness)) {
	case "ready", "waiting_for_finger":
		return true
	default:
		return false
	}
}

// Reader readiness values reported by the ESP32. Unknown values are rejected so
// the admin UI never has to render an arbitrary sensor string.
var deviceReaderReadinessValues = map[string]bool{
	"not_ready":          true,
	"initializing":       true,
	"ready":              true,
	"waiting_for_finger": true,
	"finger_detected":    true,
	"capture_failed":     true,
	"enrolling":          true,
}

func normalizeDeviceReaderReadiness(value string) string {
	readiness := strings.ToLower(strings.TrimSpace(value))
	if deviceReaderReadinessValues[readiness] {
		return readiness
	}
	return "not_ready"
}

func (s *Server) reportDeviceReaders(c *gin.Context) {
	device := currentDevice(c)
	if device == nil {
		fail(c, http.StatusUnauthorized, "Invalid or inactive device")
		return
	}
	var request struct {
		Readers []struct {
			Key       string `json:"key"`
			Label     string `json:"label"`
			UART      int    `json:"uart"`
			Connected bool   `json:"connected"`
			Readiness string `json:"readiness"`
		} `json:"readers"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if len(request.Readers) > 8 {
		fail(c, http.StatusUnprocessableEntity, "Too many readers reported")
		return
	}

	now := time.Now().UTC()
	readers := make([]models.DeviceReader, 0, len(request.Readers))
	seen := make(map[string]bool, len(request.Readers))
	for _, item := range request.Readers {
		key := normalizeEnrollmentReaderKey(item.Key)
		if !isValidEnrollmentReaderKey(key) {
			fail(c, http.StatusUnprocessableEntity, "Unknown reader key")
			return
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		label := strings.TrimSpace(item.Label)
		defaultLabel, defaultUART := enrollmentReaderDefaults(key)
		if label == "" {
			label = defaultLabel
		}
		if len(label) > 64 {
			label = label[:64]
		}
		uart := item.UART
		if uart == 0 {
			uart = defaultUART
		}
		lastSeen := now
		readers = append(readers, models.DeviceReader{
			DeviceRegistrationID: device.ID,
			ReaderKey:            key,
			Label:                label,
			UART:                 uart,
			Connected:            item.Connected,
			Readiness:            normalizeDeviceReaderReadiness(item.Readiness),
			LastSeenAt:           &lastSeen,
			UpdatedAt:            now,
		})
	}

	readerKeys := make([]string, 0, len(readers))
	for i := range readers {
		readerKeys = append(readerKeys, readers[i].ReaderKey)
	}
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		for i := range readers {
			reader := readers[i]
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "device_registration_id"}, {Name: "reader_key"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"label", "uart", "connected", "readiness", "last_seen_at", "updated_at",
				}),
			}).Create(&reader).Error; err != nil {
				return err
			}
		}
		query := tx.Where("device_registration_id = ?", device.ID)
		if len(readerKeys) > 0 {
			query = query.Where("reader_key NOT IN ?", readerKeys)
		}
		if err := query.Delete(&models.DeviceReader{}).Error; err != nil {
			return err
		}
		return tx.Model(&models.DeviceRegistration{}).
			Where("id = ?", device.ID).
			Update("last_seen_at", now).Error
	})
	if err != nil {
		s.Log.Error("device reader report failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to report reader status")
		return
	}

	response := make([]gin.H, 0, len(readers))
	for i := range readers {
		response = append(response, serializeEnrollmentReader(readers[i], readers[i].Connected))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Reader status reported",
		"readers": response,
	})
}

func (s *Server) memberEnrollmentReaders(c *gin.Context) {
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

	requestedID := int64(0)
	if raw := strings.TrimSpace(c.Query("device_registration_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			fail(c, http.StatusUnprocessableEntity, "Invalid device_registration_id")
			return
		}
		requestedID = parsed
	}

	session, err := s.latestEnrollmentSession(ctx, member.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.Log.Error("enrollment session lookup failed", "member_id", member.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load enrollment readers")
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		session = nil
	}

	var device *models.DeviceRegistration
	if session != nil && session.IsActive && session.DeviceRegistrationID != nil {
		device = s.loadEnrollmentDevice(ctx, session.DeviceRegistrationID)
	}
	if device == nil {
		device, err = s.resolveEnrollmentDevice(ctx, requestedID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				fail(c, http.StatusConflict, "No active biometric device is available for enrollment")
				return
			}
			s.Log.Error("enrollment device lookup failed", "error", err)
			fail(c, http.StatusInternalServerError, "Unable to load enrollment readers")
			return
		}
	}

	readers, err := s.enrollmentReadersForDevice(ctx, device)
	if err != nil {
		s.Log.Error("enrollment reader lookup failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load enrollment readers")
		return
	}

	payload := gin.H{
		"success":             true,
		"member_id":           member.ID,
		"device":              serializeDevice(device),
		"readers":             readers,
		"selected_reader_key": nil,
	}
	if session != nil {
		payload["session_uid"] = session.SessionUID
		payload["selected_reader_key"] = session.ReaderKey
	}
	c.JSON(http.StatusOK, payload)
}

func (s *Server) enrollmentReadersForDevice(ctx context.Context, device *models.DeviceRegistration) ([]gin.H, error) {
	var reported []models.DeviceReader
	if err := s.DB.WithContext(ctx).
		Where("device_registration_id = ?", device.ID).
		Find(&reported).Error; err != nil {
		return nil, err
	}
	byKey := make(map[string]models.DeviceReader, len(reported))
	for i := range reported {
		byKey[reported[i].ReaderKey] = reported[i]
	}

	now := time.Now().UTC()
	items := make([]gin.H, 0, 3)
	for _, key := range []string{enrollmentReaderLogin, enrollmentReaderLogout} {
		defaultLabel, defaultUART := enrollmentReaderDefaults(key)
		reader, found := byKey[key]
		if !found {
			reader = models.DeviceReader{
				DeviceRegistrationID: device.ID,
				ReaderKey:            key,
				Label:                defaultLabel,
				UART:                 defaultUART,
			}
		}
		available := reader.Connected && enrollmentReaderIsFresh(reader.LastSeenAt, now) &&
			enrollmentReaderCanStart(reader.Readiness)
		items = append(items, serializeEnrollmentReader(reader, available))
	}

	// A dedicated reader exists only on a board where it is compiled in. The
	// admin UI must not offer it while it is disconnected.
	if reader, found := byKey[enrollmentReaderDedicated]; found {
		available := reader.Connected && enrollmentReaderIsFresh(reader.LastSeenAt, now) &&
			enrollmentReaderCanStart(reader.Readiness)
		if available {
			items = append(items, serializeEnrollmentReader(reader, true))
		}
	}
	s.Log.Info("enrollment readers loaded",
		"device_id", device.ID,
		"device_online", deviceIsOnline(device.LastSeenAt, now),
		"readers", items)
	return items, nil
}

func (s *Server) enrollmentReaderAvailable(ctx context.Context, deviceID int64, readerKey string, now time.Time) (bool, error) {
	var reader models.DeviceReader
	err := s.DB.WithContext(ctx).
		Where("device_registration_id = ? AND reader_key = ?", deviceID, readerKey).
		First(&reader).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.Log.Info("enrollment reader check", "device_id", deviceID, "reader_key", readerKey, "found", false)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	available := reader.Connected && enrollmentReaderIsFresh(reader.LastSeenAt, now) &&
		enrollmentReaderCanStart(reader.Readiness)
	s.Log.Info("enrollment reader check",
		"device_id", deviceID,
		"reader_key", readerKey,
		"found", true,
		"connected", reader.Connected,
		"readiness", reader.Readiness,
		"last_seen_at", reader.LastSeenAt,
		"fresh", enrollmentReaderIsFresh(reader.LastSeenAt, now),
		"can_start", enrollmentReaderCanStart(reader.Readiness),
		"available", available)
	return available, nil
}

func (s *Server) loadActiveEnrollmentForDevice(ctx context.Context, deviceID int64) (*models.MemberEnrollmentSession, error) {
	var session models.MemberEnrollmentSession
	err := s.DB.WithContext(ctx).
		Where("device_registration_id = ? AND is_active = true", deviceID).
		Order("id DESC").
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}
