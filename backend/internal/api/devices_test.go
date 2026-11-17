package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func TestGenerateDeviceAPIKeyIsRandomHex(t *testing.T) {
	first, err := generateDeviceAPIKey()
	if err != nil {
		t.Fatalf("generateDeviceAPIKey: %v", err)
	}
	second, err := generateDeviceAPIKey()
	if err != nil {
		t.Fatalf("generateDeviceAPIKey: %v", err)
	}
	if len(first) != 64 {
		t.Fatalf("expected 64 hex characters, got %d", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Fatalf("key is not hex: %v", err)
	}
	if first == second {
		t.Fatal("two generated keys must differ")
	}
	if hashDeviceAPIKey(first) == first {
		t.Fatal("stored digest must differ from the raw key")
	}
	if hashDeviceAPIKey("  "+first+"  ") != hashDeviceAPIKey(first) {
		t.Fatal("digest must trim surrounding whitespace like device auth does")
	}
}

func TestDeviceTokenRoundTripAndVersion(t *testing.T) {
	server := &Server{Config: config.Config{DeviceHMACSecret: "unit-test-device-secret"}}

	token := server.issueDeviceToken(42, 3)
	deviceID, version, ok := server.parseDeviceToken(token)
	if !ok || deviceID != 42 || version != 3 {
		t.Fatalf("unexpected parse result: id=%d version=%d ok=%v", deviceID, version, ok)
	}

	other := &Server{Config: config.Config{DeviceHMACSecret: "different-secret"}}
	if _, _, ok := other.parseDeviceToken(token); ok {
		t.Fatal("token signed with another secret must be rejected")
	}

	parts := strings.Split(token, ".")
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode token payload: %v", err)
	}
	tamperedPayload := strings.Replace(string(payload), "42|", "43|", 1)
	tampered := base64.RawURLEncoding.EncodeToString([]byte(tamperedPayload)) + "." + parts[1]
	if _, _, ok := server.parseDeviceToken(tampered); ok {
		t.Fatal("tampered token must be rejected")
	}
}

func TestLegacyTwoPartDeviceTokenParsesAsVersionZero(t *testing.T) {
	secret := "unit-test-device-secret"
	server := &Server{Config: config.Config{DeviceHMACSecret: secret}}
	expires := time.Now().Add(time.Hour).Unix()
	payload := fmt.Sprintf("%d|%d", 7, expires)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	deviceID, version, ok := server.parseDeviceToken(token)
	if !ok || deviceID != 7 || version != 0 {
		t.Fatalf("legacy token parse: id=%d version=%d ok=%v", deviceID, version, ok)
	}
}

func TestSerializeDeviceOmitsSecrets(t *testing.T) {
	device := models.DeviceRegistration{
		ID:           5,
		DeviceUID:    "esp32-0123456789abcdef",
		Name:         "ALF V1.0 ESP32",
		DeviceType:   "esp32",
		APIKeyDigest: "super-secret-digest",
		Status:       "active",
		TokenVersion: 9,
	}
	serialized := serializeDevice(&device)
	for _, forbidden := range []string{"api_key", "api_key_digest", "token_version"} {
		if _, exists := serialized[forbidden]; exists {
			t.Fatalf("serialized device must not contain %q", forbidden)
		}
	}
	if serialized["device_uid"] != device.DeviceUID {
		t.Fatalf("expected device_uid %q, got %v", device.DeviceUID, serialized["device_uid"])
	}
	if serialized["status"] != "active" {
		t.Fatalf("expected active status, got %v", serialized["status"])
	}
}

func TestDeviceIsOnlineUsesStaleThreshold(t *testing.T) {
	now := time.Now().UTC()
	fresh := now.Add(-30 * time.Second)
	stale := now.Add(-deviceHeartbeatStaleAfter - time.Second)

	if !deviceIsOnline(&fresh, now) {
		t.Fatal("a 30s-old heartbeat must count as online")
	}
	if deviceIsOnline(&stale, now) {
		t.Fatal("a heartbeat older than the stale threshold must count as offline")
	}
	if deviceIsOnline(nil, now) {
		t.Fatal("a device that has never reported must not count as online")
	}
}

func TestNormalizeDeviceReaderReadiness(t *testing.T) {
	cases := map[string]string{
		"ready":              "ready",
		"  ENROLLING ":       "enrolling",
		"waiting_for_finger": "waiting_for_finger",
		"capture_failed":     "capture_failed",
		"":                   "not_ready",
		"unexpected":         "not_ready",
	}
	for input, expected := range cases {
		if got := normalizeDeviceReaderReadiness(input); got != expected {
			t.Fatalf("normalizeDeviceReaderReadiness(%q) = %q, want %q", input, got, expected)
		}
	}
}
