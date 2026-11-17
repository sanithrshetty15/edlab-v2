package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const (
	deviceStatusActive  = "active"
	deviceStatusRevoked = "revoked"

	// The firmware sends a heartbeat every 30s; three missed beats means the
	// controller is offline. A stale database row must never read as online.
	deviceHeartbeatStaleAfter = 90 * time.Second
)

func deviceIsOnline(lastSeen *time.Time, now time.Time) bool {
	return lastSeen != nil && !lastSeen.IsZero() && now.Sub(lastSeen.UTC()) <= deviceHeartbeatStaleAfter
}

var (
	deviceTypePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)
	deviceUIDPattern  = regexp.MustCompile(`^[A-Za-z0-9._:-]{4,64}$`)
)

type provisionDeviceRequest struct {
	Name       string `json:"name"`
	DeviceType string `json:"device_type"`
	DeviceUID  string `json:"device_uid"`
}

func (s *Server) registerDeviceRoutes(api *gin.RouterGroup) {
	devices := api.Group("/devices")
	devices.Use(s.RequireAuth(), s.RequireAdmin())
	devices.POST("", s.provisionDevice)
	devices.GET("", s.listDevices)
	devices.GET("/:id", s.showDevice)
	devices.POST("/:id/rotate-key", s.rotateDeviceKey)
	devices.POST("/:id/revoke", s.revokeDevice)
}

// generateDeviceAPIKey returns a 256-bit random key encoded as lowercase hex.
// The raw key is returned to the caller exactly once and never persisted.
func generateDeviceAPIKey() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

// hashDeviceAPIKey hashes a raw device key exactly as device authentication
// does: SHA-256 over the trimmed key, encoded as lowercase hex.
func hashDeviceAPIKey(apiKey string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(apiKey)))
	return hex.EncodeToString(digest[:])
}

func generateDeviceUID() (string, error) {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "esp32-" + hex.EncodeToString(buffer), nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key") || strings.Contains(message, "unique constraint")
}

// serializeDevice deliberately lists only non-secret fields. Never add
// api_key_digest or token_version here.
func serializeDevice(device *models.DeviceRegistration) gin.H {
	result := gin.H{
		"id":           device.ID,
		"device_uid":   device.DeviceUID,
		"name":         device.Name,
		"device_type":  device.DeviceType,
		"status":       device.Status,
		"online":       deviceIsOnline(device.LastSeenAt, time.Now().UTC()),
		"last_seen_at": device.LastSeenAt,
		"created_at":   device.CreatedAt,
		"updated_at":   device.UpdatedAt,
	}
	if device.RevokedAt != nil {
		result["revoked_at"] = device.RevokedAt
	}
	return result
}

func (s *Server) provisionDevice(c *gin.Context) {
	var request provisionDeviceRequest
	if !bindJSON(c, &request) {
		return
	}

	name := strings.TrimSpace(request.Name)
	deviceType := strings.ToLower(strings.TrimSpace(request.DeviceType))
	if deviceType == "" {
		deviceType = "esp32"
	}
	providedUID := strings.TrimSpace(request.DeviceUID)

	validationErrors := []string{}
	if name == "" {
		validationErrors = append(validationErrors, "Device name is required")
	} else if len(name) > 120 {
		validationErrors = append(validationErrors, "Device name must be at most 120 characters")
	}
	if !deviceTypePattern.MatchString(deviceType) {
		validationErrors = append(validationErrors, "device_type must be 1-40 lowercase letters, digits, hyphens or underscores")
	}
	if providedUID != "" && !deviceUIDPattern.MatchString(providedUID) {
		validationErrors = append(validationErrors, "device_uid must be 4-64 characters using letters, digits, dot, underscore, colon or hyphen")
	}
	if len(validationErrors) > 0 {
		failValidation(c, strings.Join(validationErrors, ", "), validationErrors)
		return
	}

	apiKey, err := generateDeviceAPIKey()
	if err != nil {
		s.Log.Error("device api key generation failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to provision device")
		return
	}

	var device models.DeviceRegistration
	created := false
	for attempt := 0; attempt < 3; attempt++ {
		deviceUID := providedUID
		if deviceUID == "" {
			deviceUID, err = generateDeviceUID()
			if err != nil {
				s.Log.Error("device uid generation failed", "error", err)
				fail(c, http.StatusInternalServerError, "Unable to provision device")
				return
			}
		}

		device = models.DeviceRegistration{
			DeviceUID:    deviceUID,
			Name:         name,
			DeviceType:   deviceType,
			APIKeyDigest: hashDeviceAPIKey(apiKey),
			Status:       deviceStatusActive,
		}
		err = s.DB.WithContext(c.Request.Context()).Create(&device).Error
		if err == nil {
			created = true
			break
		}
		if !isDuplicateKeyError(err) {
			s.Log.Error("device provisioning failed", "error", err)
			fail(c, http.StatusInternalServerError, "Unable to provision device")
			return
		}
		if providedUID != "" {
			fail(c, http.StatusConflict, "A device with this device_uid already exists")
			return
		}
	}
	if !created {
		s.Log.Error("device provisioning failed after uid retries", "error", err)
		fail(c, http.StatusConflict, "Unable to allocate a unique device_uid; please retry")
		return
	}

	s.audit(c, "device_provisioned", "DeviceRegistration", &device.ID, gin.H{
		"device_uid":  device.DeviceUID,
		"name":        device.Name,
		"device_type": device.DeviceType,
	})

	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Device provisioned successfully. Store the API key now; it will not be shown again.",
		"device":  serializeDevice(&device),
		"api_key": apiKey,
	})
}

func (s *Server) listDevices(c *gin.Context) {
	ctx := c.Request.Context()
	var devices []models.DeviceRegistration
	if err := s.DB.WithContext(ctx).
		Order("created_at DESC, id DESC").Find(&devices).Error; err != nil {
		s.Log.Error("device list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load devices")
		return
	}

	now := time.Now().UTC()
	items := make([]gin.H, 0, len(devices))
	if len(devices) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success":                  true,
			"devices":                  items,
			"server_time":              now,
			"online_threshold_seconds": int(deviceHeartbeatStaleAfter.Seconds()),
		})
		return
	}

	deviceIDs := make([]int64, 0, len(devices))
	for i := range devices {
		deviceIDs = append(deviceIDs, devices[i].ID)
	}

	// One query per relation keeps the 4-second admin poll cheap.
	var readers []models.DeviceReader
	if err := s.DB.WithContext(ctx).
		Where("device_registration_id IN ?", deviceIDs).
		Order("reader_key ASC").Find(&readers).Error; err != nil {
		s.Log.Error("device reader list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load devices")
		return
	}
	readersByDevice := make(map[int64][]gin.H, len(devices))
	for i := range readers {
		reader := readers[i]
		available := reader.Connected && enrollmentReaderIsFresh(reader.LastSeenAt, now)
		readersByDevice[reader.DeviceRegistrationID] = append(
			readersByDevice[reader.DeviceRegistrationID],
			serializeEnrollmentReader(reader, available),
		)
	}

	var sessions []models.MemberEnrollmentSession
	if err := s.DB.WithContext(ctx).
		Where("device_registration_id IN ? AND is_active = true", deviceIDs).
		Order("id ASC").Find(&sessions).Error; err != nil {
		s.Log.Error("device enrollment list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load devices")
		return
	}
	memberIDs := make([]int64, 0, len(sessions))
	for i := range sessions {
		memberIDs = append(memberIDs, sessions[i].MemberID)
	}
	membersByID := make(map[int64]models.Member, len(memberIDs))
	if len(memberIDs) > 0 {
		var members []models.Member
		if err := s.DB.WithContext(ctx).Find(&members, memberIDs).Error; err != nil {
			s.Log.Error("device enrollment member lookup failed", "error", err)
			fail(c, http.StatusInternalServerError, "Unable to load devices")
			return
		}
		for i := range members {
			membersByID[members[i].ID] = members[i]
		}
	}
	enrollmentByDevice := make(map[int64]gin.H, len(sessions))
	for i := range sessions {
		session := sessions[i]
		if session.DeviceRegistrationID == nil {
			continue
		}
		if _, exists := enrollmentByDevice[*session.DeviceRegistrationID]; exists {
			continue
		}
		var member *models.Member
		if loaded, ok := membersByID[session.MemberID]; ok {
			member = &loaded
		}
		enrollmentByDevice[*session.DeviceRegistrationID] = serializeEnrollmentSession(&session, member, nil)
	}

	for i := range devices {
		item := serializeDevice(&devices[i])
		if list, ok := readersByDevice[devices[i].ID]; ok {
			item["readers"] = list
		} else {
			item["readers"] = []gin.H{}
		}
		if enrollment, ok := enrollmentByDevice[devices[i].ID]; ok {
			item["enrollment"] = enrollment
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                  true,
		"devices":                  items,
		"server_time":              now,
		"online_threshold_seconds": int(deviceHeartbeatStaleAfter.Seconds()),
	})
}

func (s *Server) showDevice(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var device models.DeviceRegistration
	if err := s.DB.WithContext(c.Request.Context()).First(&device, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "Device not found")
			return
		}
		s.Log.Error("device lookup failed", "device_id", id, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load device")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "device": serializeDevice(&device)})
}

func (s *Server) rotateDeviceKey(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var device models.DeviceRegistration
	if err := s.DB.WithContext(c.Request.Context()).First(&device, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "Device not found")
			return
		}
		s.Log.Error("device lookup failed", "device_id", id, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to rotate device key")
		return
	}
	if device.Status == deviceStatusRevoked {
		fail(c, http.StatusConflict, "Revoked devices cannot rotate keys; provision a replacement device")
		return
	}

	apiKey, err := generateDeviceAPIKey()
	if err != nil {
		s.Log.Error("device api key generation failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to rotate device key")
		return
	}

	now := time.Now().UTC()
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.DeviceRegistration{}).
		Where("id = ?", device.ID).
		Updates(map[string]any{
			"api_key_digest": hashDeviceAPIKey(apiKey),
			"token_version":  gorm.Expr("token_version + 1"),
			"updated_at":     now,
		}).Error; err != nil {
		s.Log.Error("device key rotation failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to rotate device key")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).First(&device, device.ID).Error; err != nil {
		s.Log.Error("device reload failed after rotation", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to rotate device key")
		return
	}

	s.audit(c, "device_key_rotated", "DeviceRegistration", &device.ID, gin.H{
		"device_uid": device.DeviceUID,
		"name":       device.Name,
	})

	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device API key rotated successfully. Store the new key now; it will not be shown again.",
		"device":  serializeDevice(&device),
		"api_key": apiKey,
	})
}

func (s *Server) revokeDevice(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var device models.DeviceRegistration
	if err := s.DB.WithContext(c.Request.Context()).First(&device, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "Device not found")
			return
		}
		s.Log.Error("device lookup failed", "device_id", id, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to revoke device")
		return
	}
	if device.Status == deviceStatusRevoked {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Device is already revoked",
			"device":  serializeDevice(&device),
		})
		return
	}

	now := time.Now().UTC()
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.DeviceRegistration{}).
		Where("id = ?", device.ID).
		Updates(map[string]any{
			"status":        deviceStatusRevoked,
			"revoked_at":    now,
			"token_version": gorm.Expr("token_version + 1"),
			"updated_at":    now,
		}).Error; err != nil {
		s.Log.Error("device revocation failed", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to revoke device")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).First(&device, device.ID).Error; err != nil {
		s.Log.Error("device reload failed after revocation", "device_id", device.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to revoke device")
		return
	}

	s.audit(c, "device_revoked", "DeviceRegistration", &device.ID, gin.H{
		"device_uid": device.DeviceUID,
		"name":       device.Name,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device revoked successfully",
		"device":  serializeDevice(&device),
	})
}
