package api

import (
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

	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/database"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/ratelimit"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

type enrollmentTestEnv struct {
	t          *testing.T
	router     http.Handler
	db         *gorm.DB
	adminToken string
	deviceID   int64
	deviceKey  string
	counter    int
}

type enrollmentTestSession struct {
	SessionUID      string  `json:"session_uid"`
	Status          string  `json:"status"`
	MemberID        int64   `json:"member_id"`
	ReaderKey       *string `json:"reader_key"`
	FingerprintSlot *int    `json:"fingerprint_slot"`
	CleanupRequired bool    `json:"cleanup_required"`
	IsActive        bool    `json:"is_active"`
	ErrorCode       string  `json:"error_code"`
}

type enrollmentTestReader struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	UART      int    `json:"uart"`
	Connected bool   `json:"connected"`
	Available bool   `json:"available"`
}

type enrollmentTestReadersResponse struct {
	Success           bool                   `json:"success"`
	Readers           []enrollmentTestReader `json:"readers"`
	SelectedReaderKey *string                `json:"selected_reader_key"`
	Device            struct {
		ID int64 `json:"id"`
	} `json:"device"`
}

type enrollmentTestMember struct {
	ID               int64   `json:"id"`
	LoginID          string  `json:"login_id"`
	Email            string  `json:"email"`
	USN              *string `json:"usn"`
	RollNumber       *string `json:"roll_number"`
	Status           string  `json:"status"`
	EnrollmentStatus string  `json:"enrollment_status"`
}

type enrollmentTestEmail struct {
	Recipient string `json:"recipient"`
	Status    string `json:"status"`
	Attempts  int    `json:"attempts"`
	LastError string `json:"last_error"`
}

type enrollmentTestResponse struct {
	Success             bool                   `json:"success"`
	Message             string                 `json:"message"`
	Action              string                 `json:"action"`
	Member              enrollmentTestMember   `json:"member"`
	Enrollment          *enrollmentTestSession `json:"enrollment"`
	RegistrationEmail   *enrollmentTestEmail   `json:"registration_email"`
	BiometricEnrollment struct {
		ID              int64  `json:"id"`
		MemberID        int64  `json:"member_id"`
		DeviceUserID    string `json:"device_user_id"`
		FingerprintSlot *int   `json:"fingerprint_slot"`
		Status          string `json:"status"`
	} `json:"biometric_enrollment"`
}

func setupEnrollmentTestEnv(t *testing.T) *enrollmentTestEnv {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}

	cfg := config.Config{
		AppEnv:           "test",
		DatabaseURL:      dsn,
		JWTSecret:        strings.Repeat("j", 32),
		SessionSecret:    strings.Repeat("s", 32),
		DeviceHMACSecret: "enrollment-integration-device-secret",
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

	env := &enrollmentTestEnv{
		t:         t,
		router:    server.Router(),
		db:        dbHandler.DB,
		deviceKey: fmt.Sprintf("enrollment-device-key-%d", time.Now().UnixNano()),
	}

	admin := models.User{Email: fmt.Sprintf("enrollment-admin-%d@example.com", time.Now().UnixNano()), Role: "admin", Active: true}
	if err := env.db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	env.adminToken = fmt.Sprintf("enrollment-admin-session-%d", time.Now().UnixNano())
	expires := time.Now().UTC().Add(time.Hour)
	session := models.Session{
		UserID:       admin.ID,
		TokenDigest:  securityManager.TokenDigest(env.adminToken),
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    &expires,
	}
	if err := env.db.Create(&session).Error; err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	device := models.DeviceRegistration{
		DeviceUID:    fmt.Sprintf("enrollment-device-%d", time.Now().UnixNano()),
		Name:         "Enrollment Integration Reader",
		DeviceType:   "esp32",
		APIKeyDigest: hashDeviceAPIKey(env.deviceKey),
		Status:       deviceStatusActive,
	}
	if err := env.db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	env.deviceID = device.ID
	env.reportReaders()

	return env
}

func (e *enrollmentTestEnv) nextSuffix() string {
	e.counter++
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), e.counter)
}

func (e *enrollmentTestEnv) request(method, path, bearer, deviceToken string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	return deviceTestRequest(e.t, e.router, method, path, bearer, deviceToken, body)
}

func (e *enrollmentTestEnv) decode(recorder *httptest.ResponseRecorder, target any) {
	e.t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		e.t.Fatalf("decode response: %v; body = %s", err, recorder.Body.String())
	}
}

func (e *enrollmentTestEnv) expect(recorder *httptest.ResponseRecorder, want int) {
	e.t.Helper()
	if recorder.Code != want {
		e.t.Fatalf("status = %d, want %d; body = %s", recorder.Code, want, recorder.Body.String())
	}
}

func (e *enrollmentTestEnv) createPendingMember(prefix string) (int64, string, string) {
	e.t.Helper()
	suffix := e.nextSuffix()
	value := time.Now().UnixNano() % 1000000000
	loginID := fmt.Sprintf("%s-%s", prefix, suffix)
	email := fmt.Sprintf("%s-%s@example.com", prefix, suffix)
	body := map[string]any{
		"name":                  "Enrollment Test Member",
		"login_id":              loginID,
		"email":                 email,
		"phone":                 fmt.Sprintf("+919%09d", value),
		"password":              "password123",
		"password_confirmation": "password123",
		"academic_year":         2,
		"usn":                   fmt.Sprintf("TST%d", value),
		"branch":                "Computer Science",
	}
	recorder := e.request(http.MethodPost, "/api/v1/members-management", e.adminToken, "", body)
	e.expect(recorder, http.StatusCreated)
	var response enrollmentTestResponse
	e.decode(recorder, &response)
	if response.Member.Status != memberStatusPendingEnrollment {
		e.t.Fatalf("new member status = %q, want %q", response.Member.Status, memberStatusPendingEnrollment)
	}
	if response.Enrollment == nil || response.Enrollment.Status != enrollmentStatusPendingDetails {
		e.t.Fatalf("unexpected initial enrollment: %+v", response.Enrollment)
	}
	return response.Member.ID, loginID, response.Enrollment.SessionUID
}

func (e *enrollmentTestEnv) reportReaders() {
	e.t.Helper()
	recorder := e.request(http.MethodPost, "/api/v1/biometric/readers", "", e.deviceKey, map[string]any{
		"readers": []map[string]any{
			{"key": "login", "label": "Login reader", "uart": 2, "connected": true},
			{"key": "logout", "label": "Logout reader", "uart": 1, "connected": true},
		},
	})
	e.expect(recorder, http.StatusOK)
}

func (e *enrollmentTestEnv) startEnrollment(memberID int64, readerKey string) enrollmentTestSession {
	e.t.Helper()
	e.reportReaders()
	recorder := e.request(http.MethodPost, fmt.Sprintf("/api/v1/members-management/%d/fingerprint-enrollment", memberID), e.adminToken, "", map[string]any{
		"device_registration_id": e.deviceID,
		"reader_key":             readerKey,
	})
	e.expect(recorder, http.StatusOK)
	var response enrollmentTestResponse
	e.decode(recorder, &response)
	if response.Enrollment == nil || response.Enrollment.SessionUID == "" {
		e.t.Fatalf("missing enrollment session: %s", recorder.Body.String())
	}
	if response.Enrollment.Status != enrollmentStatusAwaitingSensor {
		e.t.Fatalf("started session status = %q, want %q", response.Enrollment.Status, enrollmentStatusAwaitingSensor)
	}
	if response.Enrollment.ReaderKey == nil || *response.Enrollment.ReaderKey != readerKey {
		e.t.Fatalf("started session reader = %v, want %q", response.Enrollment.ReaderKey, readerKey)
	}
	return *response.Enrollment
}

func (e *enrollmentTestEnv) progress(sessionUID, status string, slot *int, errorCode, errorMessage string) enrollmentTestSession {
	e.t.Helper()
	body := map[string]any{"status": status}
	if slot != nil {
		body["fingerprint_slot"] = *slot
	}
	if errorCode != "" {
		body["error_code"] = errorCode
	}
	if errorMessage != "" {
		body["error_message"] = errorMessage
	}
	recorder := e.request(http.MethodPost, fmt.Sprintf("/api/v1/biometric/enrollment-sessions/%s/progress", sessionUID), "", e.deviceKey, body)
	e.expect(recorder, http.StatusOK)
	var response enrollmentTestResponse
	e.decode(recorder, &response)
	if response.Enrollment == nil {
		e.t.Fatalf("missing enrollment in progress response: %s", recorder.Body.String())
	}
	return *response.Enrollment
}

func (e *enrollmentTestEnv) claim() enrollmentTestResponse {
	e.t.Helper()
	recorder := e.request(http.MethodGet, "/api/v1/biometric/enrollment-session", "", e.deviceKey, nil)
	e.expect(recorder, http.StatusOK)
	var response enrollmentTestResponse
	e.decode(recorder, &response)
	return response
}

func (e *enrollmentTestEnv) complete(sessionUID string, slot int) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.request(http.MethodPost, fmt.Sprintf("/api/v1/biometric/enrollment-sessions/%s/complete", sessionUID), "", e.deviceKey, map[string]any{
		"fingerprint_slot": slot,
		"device_user_id":   fmt.Sprintf("slot:%d", slot),
	})
}

func (e *enrollmentTestEnv) loadMember(memberID int64) models.Member {
	e.t.Helper()
	var member models.Member
	if err := e.db.First(&member, memberID).Error; err != nil {
		e.t.Fatalf("load member %d: %v", memberID, err)
	}
	return member
}

func (e *enrollmentTestEnv) loadUser(userID int64) models.User {
	e.t.Helper()
	var user models.User
	if err := e.db.First(&user, userID).Error; err != nil {
		e.t.Fatalf("load user %d: %v", userID, err)
	}
	return user
}

func TestMemberFingerprintEnrollmentIntegration(t *testing.T) {
	env := setupEnrollmentTestEnv(t)

	t.Run("successful two-scan enrollment activates only after finalization", func(t *testing.T) {
		memberID, _, _ := env.createPendingMember("enroll-success")
		member := env.loadMember(memberID)
		if user := env.loadUser(member.UserID); user.Active {
			t.Fatal("pending member user must start inactive")
		}

		recorder := env.request(http.MethodPatch, fmt.Sprintf("/api/v1/members-management/%d/status", memberID), env.adminToken, "", map[string]any{"status": "active"})
		env.expect(recorder, http.StatusUnprocessableEntity)
		recorder = env.request(http.MethodDelete, fmt.Sprintf("/api/v1/members-management/%d", memberID), env.adminToken, "", nil)
		env.expect(recorder, http.StatusUnprocessableEntity)

		session := env.startEnrollment(memberID, "login")
		claimed := env.claim()
		if claimed.Action != "enroll" || claimed.Enrollment == nil || claimed.Enrollment.SessionUID != session.SessionUID {
			t.Fatalf("unexpected claim: %+v", claimed)
		}
		env.progress(session.SessionUID, enrollmentStatusWaitingFirstScan, nil, "", "")
		env.progress(session.SessionUID, enrollmentStatusFirstScan, nil, "", "")
		env.progress(session.SessionUID, enrollmentStatusWaitingRemoval, nil, "", "")
		env.progress(session.SessionUID, enrollmentStatusWaitingSecond, nil, "", "")
		env.progress(session.SessionUID, enrollmentStatusSecondScan, nil, "", "")
		slot := 7
		env.progress(session.SessionUID, enrollmentStatusVerifying, &slot, "", "")

		recorder = env.complete(session.SessionUID, slot)
		env.expect(recorder, http.StatusOK)
		var response enrollmentTestResponse
		env.decode(recorder, &response)
		if response.Member.Status != "active" {
			t.Fatalf("member status after completion = %q, want active", response.Member.Status)
		}
		if response.Enrollment == nil || response.Enrollment.Status != enrollmentStatusFinalized {
			t.Fatalf("unexpected finalized session: %+v", response.Enrollment)
		}
		if response.RegistrationEmail == nil || response.RegistrationEmail.Status != registrationEmailFailed {
			t.Fatalf("unexpected registration email after finalization: %+v", response.RegistrationEmail)
		}
		if response.BiometricEnrollment.FingerprintSlot == nil || *response.BiometricEnrollment.FingerprintSlot != slot {
			t.Fatalf("missing biometric slot in response: %s", recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "fingerprint_template") {
			t.Fatal("completion response must not expose fingerprint templates")
		}

		member = env.loadMember(memberID)
		if member.Status != "active" {
			t.Fatalf("stored member status = %q, want active", member.Status)
		}
		if user := env.loadUser(member.UserID); !user.Active {
			t.Fatal("stored user must be active after finalization")
		}
		var enrollment models.BiometricEnrollment
		if err := env.db.Where("member_id = ? AND device_registration_id = ?", memberID, env.deviceID).First(&enrollment).Error; err != nil {
			t.Fatalf("load biometric enrollment: %v", err)
		}
		if enrollment.FingerprintSlot == nil || *enrollment.FingerprintSlot != slot {
			t.Fatalf("stored slot = %v, want %d", enrollment.FingerprintSlot, slot)
		}
		if enrollment.DeviceUserID != fmt.Sprintf("slot:%d", slot) {
			t.Fatalf("stored device_user_id = %q", enrollment.DeviceUserID)
		}

		claim := env.claim()
		if claim.Action != "none" || claim.Enrollment != nil {
			t.Fatalf("finalized session must not be claimable: %+v", claim)
		}
		recorder = env.request(http.MethodGet, fmt.Sprintf("/api/v1/biometric/enrollment-sessions/%s", session.SessionUID), "", env.deviceKey, nil)
		env.expect(recorder, http.StatusOK)
		var detail enrollmentTestResponse
		env.decode(recorder, &detail)
		if detail.Enrollment == nil || detail.Enrollment.Status != enrollmentStatusFinalized {
			t.Fatalf("device detail status = %+v", detail.Enrollment)
		}
		recorder = env.request(http.MethodGet, "/api/v1/biometric/enrollments", "", env.deviceKey, nil)
		env.expect(recorder, http.StatusOK)
		if !strings.Contains(recorder.Body.String(), fmt.Sprintf(`"fingerprint_slot":%d`, slot)) {
			t.Fatalf("enrollment list missing slot: %s", recorder.Body.String())
		}

		recorder = env.request(http.MethodGet, fmt.Sprintf("/api/v1/members-management/%d/registration-email", memberID), env.adminToken, "", nil)
		env.expect(recorder, http.StatusOK)
		var emailResponse struct {
			RegistrationEmail *enrollmentTestEmail `json:"registration_email"`
		}
		env.decode(recorder, &emailResponse)
		if emailResponse.RegistrationEmail == nil || emailResponse.RegistrationEmail.Status != registrationEmailFailed {
			t.Fatalf("unexpected confirmation email status: %s", recorder.Body.String())
		}
	})

	t.Run("failed enrollment keeps member pending and supports retry", func(t *testing.T) {
		memberID, _, _ := env.createPendingMember("enroll-retry")
		session := env.startEnrollment(memberID, "login")
		slot := 8
		failed := env.progress(session.SessionUID, enrollmentStatusFailed, &slot, "duplicate_fingerprint", "Fingerprint already belongs to another member")
		if failed.Status != enrollmentStatusFailed || !failed.CleanupRequired {
			t.Fatalf("unexpected failed session: %+v", failed)
		}
		member := env.loadMember(memberID)
		if member.Status != memberStatusPendingEnrollment {
			t.Fatalf("member status after failure = %q, want pending", member.Status)
		}
		if user := env.loadUser(member.UserID); user.Active {
			t.Fatal("user must remain inactive after failed enrollment")
		}
		var mappingCount int64
		env.db.Model(&models.BiometricEnrollment{}).Where("member_id = ?", memberID).Count(&mappingCount)
		if mappingCount != 0 {
			t.Fatal("failed enrollment must not create a biometric mapping")
		}

		claim := env.claim()
		if claim.Action != "cleanup" || claim.Enrollment == nil || claim.Enrollment.SessionUID != session.SessionUID {
			t.Fatalf("unexpected cleanup claim: %+v", claim)
		}
		recorder := env.request(http.MethodPost, fmt.Sprintf("/api/v1/biometric/enrollment-sessions/%s/cleanup", session.SessionUID), "", env.deviceKey, map[string]any{"fingerprint_slot": slot})
		env.expect(recorder, http.StatusOK)
		if claim := env.claim(); claim.Action != "none" {
			t.Fatalf("cleanup was not acknowledged: %+v", claim)
		}

		retry := env.startEnrollment(memberID, "login")
		if retry.SessionUID == session.SessionUID || retry.Status != enrollmentStatusAwaitingSensor {
			t.Fatalf("unexpected retry session: %+v", retry)
		}
		newSlot := 9
		env.progress(retry.SessionUID, enrollmentStatusVerifying, &newSlot, "", "")
		recorder = env.complete(retry.SessionUID, newSlot)
		env.expect(recorder, http.StatusOK)
		if member := env.loadMember(memberID); member.Status != "active" {
			t.Fatalf("retry completion status = %q, want active", member.Status)
		}
	})

	t.Run("slot conflict does not delete another member template or activate", func(t *testing.T) {
		memberID, _, _ := env.createPendingMember("enroll-conflict")
		session := env.startEnrollment(memberID, "login")
		slot := 7 // Already finalized by the first subtest.
		env.progress(session.SessionUID, enrollmentStatusVerifying, &slot, "", "")
		recorder := env.complete(session.SessionUID, slot)
		env.expect(recorder, http.StatusConflict)
		member := env.loadMember(memberID)
		if member.Status != memberStatusPendingEnrollment {
			t.Fatalf("slot-conflict member status = %q, want pending", member.Status)
		}
		var mappingCount int64
		env.db.Model(&models.BiometricEnrollment{}).Where("member_id = ?", memberID).Count(&mappingCount)
		if mappingCount != 0 {
			t.Fatal("slot conflict must not create a mapping")
		}
		var stored models.MemberEnrollmentSession
		if err := env.db.Where("session_uid = ?", session.SessionUID).First(&stored).Error; err != nil {
			t.Fatalf("load conflict session: %v", err)
		}
		if stored.Status != enrollmentStatusFailed {
			t.Fatalf("conflict session status = %q, want failed", stored.Status)
		}
		if stored.CleanupRequired {
			t.Fatal("slot conflicts must not request cleanup of another member's template")
		}
	})

	t.Run("sensor failure and mismatched completion slot remain pending", func(t *testing.T) {
		memberID, _, _ := env.createPendingMember("enroll-sensor-fail")
		session := env.startEnrollment(memberID, "login")
		claimed := env.claim()
		if claimed.Action != "enroll" {
			t.Fatalf("expected enroll claim, got %+v", claimed)
		}
		slot := 10
		env.progress(session.SessionUID, enrollmentStatusVerifying, &slot, "", "")
		recorder := env.complete(session.SessionUID, 11)
		env.expect(recorder, http.StatusConflict)
		if member := env.loadMember(memberID); member.Status != memberStatusPendingEnrollment {
			t.Fatalf("mismatched completion activated member: %q", member.Status)
		}
		failed := env.progress(session.SessionUID, enrollmentStatusFailed, nil, "sensor_disconnected", "R307S did not respond")
		if failed.Status != enrollmentStatusFailed || failed.ErrorCode != "sensor_disconnected" {
			t.Fatalf("unexpected sensor failure session: %+v", failed)
		}
		if !failed.CleanupRequired {
			t.Fatal("sensor failure after slot allocation must require cleanup")
		}
		recorder = env.request(http.MethodPost, fmt.Sprintf("/api/v1/biometric/enrollment-sessions/%s/cleanup", session.SessionUID), "", env.deviceKey, map[string]any{"fingerprint_slot": slot})
		env.expect(recorder, http.StatusOK)
	})

	t.Run("expired session remains pending and cleans up allocated slot", func(t *testing.T) {
		memberID, _, _ := env.createPendingMember("enroll-timeout")
		session := env.startEnrollment(memberID, "login")
		slot := 11
		env.progress(session.SessionUID, enrollmentStatusVerifying, &slot, "", "")
		past := time.Now().UTC().Add(-time.Minute)
		if err := env.db.Model(&models.MemberEnrollmentSession{}).Where("session_uid = ?", session.SessionUID).Update("expires_at", past).Error; err != nil {
			t.Fatalf("expire session: %v", err)
		}

		recorder := env.request(http.MethodGet, fmt.Sprintf("/api/v1/members-management/%d/fingerprint-enrollment", memberID), env.adminToken, "", nil)
		env.expect(recorder, http.StatusOK)
		var response enrollmentTestResponse
		env.decode(recorder, &response)
		if response.Enrollment == nil || response.Enrollment.Status != enrollmentStatusExpired {
			t.Fatalf("unexpected expired status: %+v", response.Enrollment)
		}
		if !response.Enrollment.CleanupRequired {
			t.Fatal("expired session with an allocated slot must require cleanup")
		}
		if member := env.loadMember(memberID); member.Status != memberStatusPendingEnrollment {
			t.Fatalf("expired session activated member: %q", member.Status)
		}

		claim := env.claim()
		if claim.Action != "cleanup" {
			t.Fatalf("expected cleanup claim, got %+v", claim)
		}
		recorder = env.request(http.MethodPost, fmt.Sprintf("/api/v1/biometric/enrollment-sessions/%s/cleanup", session.SessionUID), "", env.deviceKey, map[string]any{"fingerprint_slot": slot})
		env.expect(recorder, http.StatusOK)
	})

	t.Run("cancellation removes pending registration and cleans up slot", func(t *testing.T) {
		memberID, loginID, _ := env.createPendingMember("enroll-cancel")
		member := env.loadMember(memberID)
		session := env.startEnrollment(memberID, "login")
		slot := 12
		env.progress(session.SessionUID, enrollmentStatusVerifying, &slot, "", "")

		recorder := env.request(http.MethodDelete, fmt.Sprintf("/api/v1/members-management/%d/fingerprint-enrollment", memberID), env.adminToken, "", nil)
		env.expect(recorder, http.StatusOK)
		var response enrollmentTestResponse
		env.decode(recorder, &response)
		if response.Enrollment == nil || response.Enrollment.Status != enrollmentStatusCancelled || !response.Enrollment.CleanupRequired {
			t.Fatalf("unexpected cancel response: %s", recorder.Body.String())
		}

		var memberCount int64
		env.db.Model(&models.Member{}).Where("id = ?", memberID).Count(&memberCount)
		if memberCount != 0 {
			t.Fatal("cancelled pending member was not deleted")
		}
		var userCount int64
		env.db.Model(&models.User{}).Where("id = ?", member.UserID).Count(&userCount)
		if userCount != 0 {
			t.Fatal("cancelled pending user was not deleted")
		}
		var loginCount int64
		env.db.Model(&models.Member{}).Where("login_id = ?", loginID).Count(&loginCount)
		if loginCount != 0 {
			t.Fatal("cancelled login ID was not released")
		}

		claim := env.claim()
		if claim.Action != "cleanup" || claim.Enrollment == nil || claim.Enrollment.SessionUID != session.SessionUID {
			t.Fatalf("unexpected cancel cleanup claim: %+v", claim)
		}
		recorder = env.request(http.MethodPost, fmt.Sprintf("/api/v1/biometric/enrollment-sessions/%s/cleanup", session.SessionUID), "", env.deviceKey, map[string]any{"fingerprint_slot": slot})
		env.expect(recorder, http.StatusOK)
		if claim := env.claim(); claim.Action != "none" {
			t.Fatalf("cancel cleanup still pending: %+v", claim)
		}
	})

	t.Run("simplified registration accepts the four required fields", func(t *testing.T) {
		value := time.Now().UnixNano() % 1000000000
		suffix := env.nextSuffix()
		roll := fmt.Sprintf("R%d", value)
		firstYearEmail := fmt.Sprintf("first-year-%s@example.com", suffix)
		body := map[string]any{
			"name":          "First Year Member",
			"academic_year": 1,
			"roll_number":   roll,
			"email":         firstYearEmail,
		}
		recorder := env.request(http.MethodPost, "/api/v1/members-management", env.adminToken, "", body)
		env.expect(recorder, http.StatusCreated)
		var response enrollmentTestResponse
		env.decode(recorder, &response)
		if response.Member.LoginID != roll {
			t.Fatalf("generated login ID = %q, want roll %q", response.Member.LoginID, roll)
		}
		if response.Member.RollNumber == nil || *response.Member.RollNumber != roll || response.Member.USN != nil {
			t.Fatalf("unexpected first-year identity: %s", recorder.Body.String())
		}
		if response.Member.Status != memberStatusPendingEnrollment || response.Enrollment == nil {
			t.Fatalf("first-year member must await enrollment: %s", recorder.Body.String())
		}

		recorder = env.request(http.MethodPost, "/api/v1/members-management", env.adminToken, "", body)
		env.expect(recorder, http.StatusUnprocessableEntity)

		usn := fmt.Sprintf("TST%d", value+1)
		secondYearEmail := fmt.Sprintf("second-year-%s@example.com", suffix)
		recorder = env.request(http.MethodPost, "/api/v1/members-management", env.adminToken, "", map[string]any{
			"name":          "Second Year Member",
			"academic_year": 3,
			"usn":           usn,
			"email":         secondYearEmail,
		})
		env.expect(recorder, http.StatusCreated)
		var secondResponse enrollmentTestResponse
		env.decode(recorder, &secondResponse)
		if secondResponse.Member.LoginID != usn {
			t.Fatalf("generated login ID = %q, want USN %q", secondResponse.Member.LoginID, usn)
		}
		if secondResponse.Member.USN == nil || *secondResponse.Member.USN != usn {
			t.Fatalf("unexpected second-year identity: %s", recorder.Body.String())
		}

		recorder = env.request(http.MethodPost, "/api/v1/members-management", env.adminToken, "", map[string]any{
			"name":          "Invalid Email Member",
			"academic_year": 2,
			"usn":           fmt.Sprintf("TST%d", value+2),
			"email":         "not-an-email",
		})
		env.expect(recorder, http.StatusUnprocessableEntity)
	})
}
