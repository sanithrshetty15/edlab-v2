package api

import (
	"bytes"
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
	"github.com/sanithrshetty15/edwinslab/backend/internal/jobs"
	"github.com/sanithrshetty15/edwinslab/backend/internal/mailer"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/ratelimit"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

type authTestEnv struct {
	server   *Server
	router   http.Handler
	db       *database.DBHandler
	security *security.Manager
}

type authTestLoginResponse struct {
	Success     bool       `json:"success"`
	Token       string     `json:"token"`
	TokenType   string     `json:"token_type"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Purpose     string     `json:"purpose"`
	EmailStatus string     `json:"email_status"`
	Message     string     `json:"message"`
}

func newAuthTestEnv(t *testing.T) *authTestEnv {
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
		DeviceHMACSecret: "integration-device-secret",
		SessionTTLMember: time.Hour,
		SessionTTLAdmin:  time.Hour,
		OTPExpiry:        5 * time.Minute,
		OTPMaxAttempts:   5,
		OTPRequestWindow: 15 * time.Minute,
		OTPMaxRequests:   5,
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
	mail := &mailer.Mailer{
		GmailUsername:    "notifications@example.com",
		GmailAppPassword: "test-app-password",
		From:             "Edwins Lab <notifications@example.com>",
		FrontendURL:      "https://edwinslab.vercel.app",
	}
	manager := jobs.New(&cfg, dbHandler.DB, logger)
	server := &Server{
		DB:          dbHandler.DB,
		Config:      cfg,
		Log:         logger,
		Security:    security.New(cfg.JWTSecret, cfg.SessionSecret),
		Mailer:      mail,
		Jobs:        manager,
		Limiter:     ratelimit.New(),
		GoogleLogin: newGoogleLoginService(cfg),
	}
	server.registerJobHandlers()

	return &authTestEnv{
		server:   server,
		router:   server.Router(),
		db:       dbHandler,
		security: server.Security,
	}
}

func authTestRequest(t *testing.T, router http.Handler, method, path, bearer string, body any) *httptest.ResponseRecorder {
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
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func authTestDecode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response (%d): %v; body = %s", recorder.Code, err, recorder.Body.String())
	}
	return value
}

func createAuthTestMember(t *testing.T, env *authTestEnv, password string) (models.User, models.Member) {
	t.Helper()
	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("auth-member-%d@example.com", suffix)
	usn := fmt.Sprintf("4AL25CS%03d", suffix%1000)
	hash, err := env.security.GenerateHash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := models.User{Email: email, Role: "member", Active: true, PasswordDigest: &hash}
	if err := env.db.DB.Create(&user).Error; err != nil {
		t.Fatalf("create member user: %v", err)
	}
	member := models.Member{
		UserID:             user.ID,
		Name:               "Auth Integration Member",
		USN:                &usn,
		LoginID:            strings.ToLower(usn),
		AcademicYear:       2026,
		Branch:             "CSE",
		Email:              email,
		Status:             "active",
		MustChangePassword: false,
	}
	if err := env.db.DB.Create(&member).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}
	return user, member
}

func createAuthTestAdmin(t *testing.T, env *authTestEnv) (models.User, models.Admin) {
	t.Helper()
	email := fmt.Sprintf("auth-admin-%d@example.com", time.Now().UnixNano())
	user := models.User{Email: email, Role: "admin", Active: true}
	if err := env.db.DB.Create(&user).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	admin := models.Admin{UserID: user.ID, Name: "Auth Integration Admin", IsPrimary: true}
	if err := env.db.DB.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	return user, admin
}

func TestMemberLoginSessionLifecycleIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	_, member := createAuthTestMember(t, env, "CorrectHorseBattery1!")

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/member-login", "", map[string]any{
		"login_type": "usn",
		"login_id":   strings.ToLower(*member.USN),
		"password":   "CorrectHorseBattery1!",
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("member login status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("member login must be non-cacheable, got %q", recorder.Header().Get("Cache-Control"))
	}
	if strings.Contains(recorder.Body.String(), "password_digest") || strings.Contains(recorder.Body.String(), "otp") {
		t.Fatalf("member login response leaked sensitive fields: %s", recorder.Body.String())
	}

	login := authTestDecode[authTestLoginResponse](t, recorder)
	if !login.Success || login.Token == "" || login.TokenType != "Bearer" {
		t.Fatalf("unexpected member login response: %s", recorder.Body.String())
	}
	if login.ExpiresAt == nil || !login.ExpiresAt.After(time.Now().UTC()) {
		t.Fatalf("member session expiry is missing or past: %+v", login.ExpiresAt)
	}

	recorder = authTestRequest(t, env.router, http.MethodGet, "/api/v1/auth/me", login.Token, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("me status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"role":"member"`) {
		t.Fatalf("me response missing member role: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), login.Token) {
		t.Fatal("me response leaked the session token")
	}

	recorder = authTestRequest(t, env.router, http.MethodDelete, "/api/v1/auth/logout", login.Token, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("logout status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	recorder = authTestRequest(t, env.router, http.MethodGet, "/api/v1/auth/me", login.Token, nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want 401; body = %s", recorder.Code, recorder.Body.String())
	}

	var sessions int64
	env.db.DB.Model(&models.Session{}).Where("token_digest = ?", env.security.TokenDigest(login.Token)).Count(&sessions)
	if sessions != 0 {
		t.Fatalf("logout left %d session rows", sessions)
	}
}

func TestMemberLoginRejectsInvalidCredentialsIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	_, member := createAuthTestMember(t, env, "CorrectHorseBattery1!")

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/member-login", "", map[string]any{
		"login_type": "usn",
		"login_id":   *member.USN,
		"password":   "wrong-password",
	})
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid password status = %d, want 401; body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "token") {
		t.Fatalf("rejected login returned a token: %s", recorder.Body.String())
	}
}

func TestAdminLoginOTPIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	user, _ := createAuthTestAdmin(t, env)

	var jobsBefore int64
	env.db.DB.Model(&models.BackgroundJob{}).Where("job_type = ?", "email_delivery").Count(&jobsBefore)

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/admin-login", "", map[string]any{
		"email": strings.ToUpper(user.Email),
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin login status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("admin login must be non-cacheable, got %q", recorder.Header().Get("Cache-Control"))
	}
	queued := authTestDecode[authTestLoginResponse](t, recorder)
	if !queued.Success || queued.Purpose != "admin_login" || queued.EmailStatus != "queued" {
		t.Fatalf("unexpected admin OTP queue response: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"token"`) {
		t.Fatalf("admin login must not return a token before OTP verification: %s", recorder.Body.String())
	}

	var otp models.OTPCode
	if err := env.db.DB.Where("user_id = ? AND purpose = ? AND used = ?", user.ID, "admin_login", false).
		Order("created_at DESC").First(&otp).Error; err != nil {
		t.Fatalf("load issued OTP: %v", err)
	}
	if otp.OTPDigest == "" || otp.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("issued OTP is invalid: %+v", otp)
	}
	var jobsAfter int64
	env.db.DB.Model(&models.BackgroundJob{}).Where("job_type = ? AND status = ?", "email_delivery", "pending").Count(&jobsAfter)
	if jobsAfter <= jobsBefore {
		t.Fatal("admin OTP request did not enqueue an email job")
	}

	const code = "2468"
	if err := env.db.DB.Model(&models.OTPCode{}).Where("id = ?", otp.ID).Updates(map[string]any{
		"otp_digest": env.security.DigestOTP(code),
		"used":       false,
		"attempts":   0,
		"expires_at": time.Now().UTC().Add(5 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("set known OTP digest: %v", err)
	}

	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/admin-verify-otp", "", map[string]any{
		"email": user.Email,
		"otp":   code,
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("verify OTP status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("OTP verification must be non-cacheable, got %q", recorder.Header().Get("Cache-Control"))
	}
	verified := authTestDecode[authTestLoginResponse](t, recorder)
	if verified.Token == "" || verified.TokenType != "Bearer" || verified.ExpiresAt == nil {
		t.Fatalf("unexpected admin verification response: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), code) || strings.Contains(recorder.Body.String(), otp.OTPDigest) {
		t.Fatalf("admin verification response leaked OTP data: %s", recorder.Body.String())
	}

	recorder = authTestRequest(t, env.router, http.MethodGet, "/api/v1/auth/me", verified.Token, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin me status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"role":"admin"`) {
		t.Fatalf("admin me response missing role: %s", recorder.Body.String())
	}

	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/admin-verify-otp", "", map[string]any{
		"email": user.Email,
		"otp":   code,
	})
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reused OTP status = %d, want 422; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAdminLoginEmailUnavailableIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	user, _ := createAuthTestAdmin(t, env)
	env.server.Mailer = &mailer.Mailer{}

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/admin-login", "", map[string]any{
		"email": user.Email,
	})
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured email status = %d, want 503; body = %s", recorder.Code, recorder.Body.String())
	}
	var otps int64
	env.db.DB.Model(&models.OTPCode{}).Where("user_id = ?", user.ID).Count(&otps)
	if otps != 0 {
		t.Fatalf("unconfigured email created %d OTP rows", otps)
	}
}

func TestAdminLoginRateLimitIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	user, _ := createAuthTestAdmin(t, env)

	recent := time.Now().UTC().Add(-time.Minute)
	for i := 0; i < env.server.Config.OTPMaxRequests; i++ {
		otp := models.OTPCode{
			UserID:         user.ID,
			OTPDigest:      env.security.DigestOTP("1111"),
			DeliveryMethod: "email",
			Purpose:        "admin_login",
			ExpiresAt:      time.Now().UTC().Add(5 * time.Minute),
			CreatedAt:      recent,
		}
		if err := env.db.DB.Create(&otp).Error; err != nil {
			t.Fatalf("seed OTP request %d: %v", i, err)
		}
	}

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/admin-login", "", map[string]any{
		"email": user.Email,
	})
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limited admin login status = %d, want 429; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestExpiredSessionIsRejectedIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	user, _ := createAuthTestMember(t, env, "CorrectHorseBattery1!")

	token := fmt.Sprintf("expired-session-%d", time.Now().UnixNano())
	past := time.Now().UTC().Add(-time.Hour)
	session := models.Session{
		UserID:       user.ID,
		TokenDigest:  env.security.TokenDigest(token),
		LastActiveAt: past,
		ExpiresAt:    &past,
	}
	if err := env.db.DB.Create(&session).Error; err != nil {
		t.Fatalf("create expired session: %v", err)
	}

	recorder := authTestRequest(t, env.router, http.MethodGet, "/api/v1/auth/me", token, nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status = %d, want 401; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestPasswordResetIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	user, member := createAuthTestMember(t, env, "OldPassword1!")

	oldToken := fmt.Sprintf("password-reset-old-session-%d", time.Now().UnixNano())
	expires := time.Now().UTC().Add(time.Hour)
	oldSession := models.Session{
		UserID:       user.ID,
		TokenDigest:  env.security.TokenDigest(oldToken),
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    &expires,
	}
	if err := env.db.DB.Create(&oldSession).Error; err != nil {
		t.Fatalf("create pre-reset session: %v", err)
	}

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/password-reset/request", "", map[string]any{
		"login_id":        member.LoginID,
		"delivery_method": "email",
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("password reset request status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `"token"`) {
		t.Fatalf("password reset request returned a token: %s", recorder.Body.String())
	}

	var otp models.OTPCode
	if err := env.db.DB.Where("user_id = ? AND purpose = ? AND used = ?", user.ID, "password_reset", false).
		Order("created_at DESC").First(&otp).Error; err != nil {
		t.Fatalf("load password reset OTP: %v", err)
	}
	const code = "1357"
	if err := env.db.DB.Model(&models.OTPCode{}).Where("id = ?", otp.ID).Updates(map[string]any{
		"otp_digest": env.security.DigestOTP(code),
		"attempts":   0,
		"expires_at": time.Now().UTC().Add(5 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("set known password reset OTP: %v", err)
	}

	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/password-reset/complete", "", map[string]any{
		"login_id":              member.LoginID,
		"otp":                   code,
		"new_password":          "NewPassword2!",
		"password_confirmation": "NewPassword2!",
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("password reset complete status = %d; body = %s", recorder.Code, recorder.Body.String())
	}

	var sessions int64
	env.db.DB.Model(&models.Session{}).Where("user_id = ?", user.ID).Count(&sessions)
	if sessions != 0 {
		t.Fatalf("password reset left %d active sessions", sessions)
	}

	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/member-login", "", map[string]any{
		"login_type": "usn",
		"login_id":   *member.USN,
		"password":   "OldPassword1!",
	})
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("old password status = %d, want 401; body = %s", recorder.Code, recorder.Body.String())
	}

	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/member-login", "", map[string]any{
		"login_type": "usn",
		"login_id":   *member.USN,
		"password":   "NewPassword2!",
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("new password status = %d; body = %s", recorder.Code, recorder.Body.String())
	}

	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/auth/password-reset/complete", "", map[string]any{
		"login_id":              member.LoginID,
		"otp":                   code,
		"new_password":          "AnotherPassword3!",
		"password_confirmation": "AnotherPassword3!",
	})
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reused password reset OTP status = %d, want 422; body = %s", recorder.Code, recorder.Body.String())
	}
}
