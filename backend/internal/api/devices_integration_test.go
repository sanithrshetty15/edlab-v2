package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/database"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/ratelimit"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

type deviceTestPayload struct {
	ID        int64      `json:"id"`
	DeviceUID string     `json:"device_uid"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	RevokedAt *time.Time `json:"revoked_at"`
}

type deviceTestProvisionResponse struct {
	Success bool              `json:"success"`
	APIKey  string            `json:"api_key"`
	Device  deviceTestPayload `json:"device"`
}

type deviceTestAuthResponse struct {
	Success bool `json:"success"`
	Device  struct {
		ID        int64  `json:"id"`
		DeviceUID string `json:"device_uid"`
		Token     string `json:"token"`
	} `json:"device"`
}

func deviceTestRequest(t *testing.T, router http.Handler, method, path, bearer, deviceToken string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	if deviceToken != "" {
		request.Header.Set("X-Device-Token", deviceToken)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// TestDeviceProvisioningLifecycleIntegration exercises the real router, GORM,
// migrations and PostgreSQL. It is skipped unless TEST_DATABASE_URL is set.
func TestDeviceProvisioningLifecycleIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}

	cfg := config.Config{
		AppEnv:           "test",
		DatabaseURL:      dsn,
		JWTSecret:        strings.Repeat("j", 32),
		SessionSecret:    strings.Repeat("s", 32),
		DeviceHMACSecret: "integration-device-secret",
		SessionTTLMember: time.Hour,
		SessionTTLAdmin:  time.Hour,
	}
	dbHandler, err := database.InitializeDB(&cfg)
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}
	t.Cleanup(func() { _ = dbHandler.CloseDB() })
	if err := dbHandler.Migrate(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	securityManager := security.New(cfg.JWTSecret, cfg.SessionSecret)
	server := &Server{
		DB:       dbHandler.DB,
		Config:   cfg,
		Log:      logger,
		Security: securityManager,
		Limiter:  ratelimit.New(),
	}
	router := server.Router()

	email := fmt.Sprintf("device-integration-%d@example.com", time.Now().UnixNano())
	admin := models.User{Email: email, Role: "admin", Active: true}
	if err := dbHandler.DB.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	adminToken := fmt.Sprintf("device-integration-session-%d", time.Now().UnixNano())
	expires := time.Now().UTC().Add(time.Hour)
	session := models.Session{
		UserID:       admin.ID,
		TokenDigest:  securityManager.TokenDigest(adminToken),
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    &expires,
	}
	if err := dbHandler.DB.Create(&session).Error; err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	// Non-admin and unauthenticated callers must not provision devices.
	member := models.User{Email: fmt.Sprintf("device-member-%d@example.com", time.Now().UnixNano()), Role: "member", Active: true}
	if err := dbHandler.DB.Create(&member).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}
	memberToken := fmt.Sprintf("device-member-session-%d", time.Now().UnixNano())
	memberSession := models.Session{
		UserID:       member.ID,
		TokenDigest:  securityManager.TokenDigest(memberToken),
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    &expires,
	}
	if err := dbHandler.DB.Create(&memberSession).Error; err != nil {
		t.Fatalf("create member session: %v", err)
	}
	var devicesBefore int64
	dbHandler.DB.Model(&models.DeviceRegistration{}).Count(&devicesBefore)
	recorder := deviceTestRequest(t, router, http.MethodPost, "/api/v1/devices", memberToken, "", map[string]any{
		"name": "Member Attempt",
	})
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("member provision status = %d, want 403; body = %s", recorder.Code, recorder.Body.String())
	}
	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/devices", "", "", map[string]any{
		"name": "Anonymous Attempt",
	})
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous provision status = %d, want 401; body = %s", recorder.Code, recorder.Body.String())
	}
	var devicesAfter int64
	dbHandler.DB.Model(&models.DeviceRegistration{}).Count(&devicesAfter)
	if devicesAfter != devicesBefore {
		t.Fatalf("rejected callers created devices: before=%d after=%d", devicesBefore, devicesAfter)
	}

	// 1. Admin provisions a device. The raw key is returned exactly once.
	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/devices", adminToken, "", map[string]any{
		"name":        "Integration ALF V1.0 ESP32",
		"device_type": "esp32",
	})
	if recorder.Code != http.StatusCreated {
		t.Fatalf("provision status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("provision response must be non-cacheable, got %q", recorder.Header().Get("Cache-Control"))
	}
	var provisioned deviceTestProvisionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &provisioned); err != nil {
		t.Fatalf("decode provision response: %v", err)
	}
	if !provisioned.Success || len(provisioned.APIKey) != 64 {
		t.Fatalf("unexpected provision response: %s", recorder.Body.String())
	}
	if provisioned.Device.ID <= 0 || provisioned.Device.DeviceUID == "" || provisioned.Device.Status != "active" {
		t.Fatalf("unexpected provisioned device: %+v", provisioned.Device)
	}
	if strings.Contains(recorder.Body.String(), "api_key_digest") || strings.Contains(recorder.Body.String(), "token_version") {
		t.Fatalf("provision response leaked internal fields: %s", recorder.Body.String())
	}

	var stored models.DeviceRegistration
	if err := dbHandler.DB.First(&stored, provisioned.Device.ID).Error; err != nil {
		t.Fatalf("load provisioned device: %v", err)
	}
	if stored.DeviceUID != provisioned.Device.DeviceUID {
		t.Fatalf("stored device_uid = %q, response = %q", stored.DeviceUID, provisioned.Device.DeviceUID)
	}
	if stored.APIKeyDigest != hashDeviceAPIKey(provisioned.APIKey) {
		t.Fatal("stored digest does not match the returned key")
	}
	if stored.APIKeyDigest == provisioned.APIKey {
		t.Fatal("raw key must not be stored")
	}
	var rawKeyMatches int64
	dbHandler.DB.Model(&models.DeviceRegistration{}).Where("api_key_digest = ?", provisioned.APIKey).Count(&rawKeyMatches)
	if rawKeyMatches != 0 {
		t.Fatal("raw key must never be stored as a digest value")
	}

	// Duplicate device_uid is rejected and detail responses stay secret-free.
	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/devices", adminToken, "", map[string]any{
		"name":       "Duplicate ESP32",
		"device_uid": provisioned.Device.DeviceUID,
	})
	if recorder.Code != http.StatusConflict {
		t.Fatalf("duplicate device_uid status = %d, want 409; body = %s", recorder.Code, recorder.Body.String())
	}
	recorder = deviceTestRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/devices/%d", provisioned.Device.ID), adminToken, "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("show device status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), provisioned.APIKey) || strings.Contains(recorder.Body.String(), stored.APIKeyDigest) {
		t.Fatal("device detail leaked a secret")
	}

	// 2. The device exchanges its key for a token and uses the middleware.
	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/biometric/authenticate-device", "", "", map[string]any{
		"api_key":     provisioned.APIKey,
		"device_name": "Integration ALF V1.0 ESP32",
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("device auth status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var authenticated deviceTestAuthResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &authenticated); err != nil {
		t.Fatalf("decode device auth: %v", err)
	}
	if authenticated.Device.Token == "" || authenticated.Device.ID != provisioned.Device.ID {
		t.Fatalf("unexpected device auth: %+v", authenticated)
	}
	if authenticated.Device.DeviceUID != provisioned.Device.DeviceUID {
		t.Fatalf("auth device_uid = %q, want %q", authenticated.Device.DeviceUID, provisioned.Device.DeviceUID)
	}

	recorder = deviceTestRequest(t, router, http.MethodGet, "/api/v1/biometric/status", "", authenticated.Device.Token, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("device status via middleware = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	// submit-event still works with a token and with HMAC only.
	tokenEventKey := fmt.Sprintf("integration-token-%d", time.Now().UnixNano())
	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/biometric/submit-event", "", authenticated.Device.Token, map[string]any{
		"device_id":       provisioned.Device.ID,
		"event_type":      "finger_mismatch",
		"idempotency_key": tokenEventKey,
		"event_timestamp": "",
		"payload":         map[string]any{},
	})
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("token submit-event = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	hmacEventKey := fmt.Sprintf("integration-hmac-%d", time.Now().UnixNano())
	canonical := fmt.Sprintf("%d|%s|%s|%s|%s", provisioned.Device.ID, "finger_mismatch", hmacEventKey, "", "")
	mac := hmac.New(sha256.New, []byte(cfg.DeviceHMACSecret))
	mac.Write([]byte(canonical))
	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/biometric/submit-event", "", "", map[string]any{
		"device_id":       provisioned.Device.ID,
		"event_type":      "finger_mismatch",
		"idempotency_key": hmacEventKey,
		"event_timestamp": "",
		"hmac_signature":  hex.EncodeToString(mac.Sum(nil)),
		"payload":         map[string]any{},
	})
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("HMAC submit-event = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	// 3. Admin listing never exposes the key or digest.
	recorder = deviceTestRequest(t, router, http.MethodGet, "/api/v1/devices", adminToken, "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list devices status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	listBody := recorder.Body.String()
	if strings.Contains(listBody, provisioned.APIKey) || strings.Contains(listBody, stored.APIKeyDigest) {
		t.Fatal("device list leaked a secret")
	}
	if strings.Contains(listBody, "api_key_digest") || strings.Contains(listBody, "token_version") {
		t.Fatal("device list leaked an internal field")
	}

	// 4. Rotation issues a new key, invalidates the old key and old token.
	recorder = deviceTestRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/rotate-key", provisioned.Device.ID), adminToken, "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("rotate status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var rotated deviceTestProvisionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("decode rotate response: %v", err)
	}
	if rotated.APIKey == "" || rotated.APIKey == provisioned.APIKey {
		t.Fatal("rotation must return a new key")
	}

	recorder = deviceTestRequest(t, router, http.MethodGet, "/api/v1/biometric/status", "", authenticated.Device.Token, nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("old token after rotation = %d, want 401", recorder.Code)
	}
	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/biometric/authenticate-device", "", "", map[string]any{
		"api_key": provisioned.APIKey,
	})
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("old key after rotation = %d, want 401", recorder.Code)
	}

	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/biometric/authenticate-device", "", "", map[string]any{
		"api_key": rotated.APIKey,
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("new key auth = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var reauthenticated deviceTestAuthResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &reauthenticated); err != nil {
		t.Fatalf("decode re-auth: %v", err)
	}
	recorder = deviceTestRequest(t, router, http.MethodGet, "/api/v1/biometric/status", "", reauthenticated.Device.Token, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("new token status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	// 5. Revocation blocks both the current token and the current key.
	recorder = deviceTestRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/revoke", provisioned.Device.ID), adminToken, "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var revoked deviceTestProvisionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &revoked); err != nil {
		t.Fatalf("decode revoke response: %v", err)
	}
	if revoked.Device.Status != "revoked" || revoked.Device.RevokedAt == nil {
		t.Fatalf("unexpected revoked device: %+v", revoked.Device)
	}
	recorder = deviceTestRequest(t, router, http.MethodGet, "/api/v1/biometric/status", "", reauthenticated.Device.Token, nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status = %d, want 401", recorder.Code)
	}
	recorder = deviceTestRequest(t, router, http.MethodPost, "/api/v1/biometric/authenticate-device", "", "", map[string]any{
		"api_key": rotated.APIKey,
	})
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("revoked key auth = %d, want 403", recorder.Code)
	}

	// 6. Audit entries exist and contain no key material.
	var audits []models.AuditLog
	if err := dbHandler.DB.Where("resource_type = ? AND resource_id = ?", "DeviceRegistration", provisioned.Device.ID).
		Order("id ASC").Find(&audits).Error; err != nil {
		t.Fatalf("load audit logs: %v", err)
	}
	actions := make([]string, 0, len(audits))
	for i := range audits {
		actions = append(actions, audits[i].Action)
		details := string(audits[i].Details)
		for _, secret := range []string{provisioned.APIKey, rotated.APIKey, stored.APIKeyDigest} {
			if strings.Contains(details, secret) {
				t.Fatalf("audit log %q leaked key material", audits[i].Action)
			}
		}
	}
	wantActions := map[string]bool{"device_provisioned": false, "device_key_rotated": false, "device_revoked": false}
	for _, action := range actions {
		if _, ok := wantActions[action]; ok {
			wantActions[action] = true
		}
	}
	for action, found := range wantActions {
		if !found {
			t.Fatalf("missing audit action %q (got %v)", action, actions)
		}
	}
}
