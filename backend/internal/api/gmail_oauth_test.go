package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/gmailapi"
	"github.com/sanithrshetty15/edwinslab/backend/internal/mailer"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

type fakeGmailOAuthService struct {
	enabled      bool
	authorized   bool
	authorizeErr error
	exchangeErr  error
	saveErr      error
	authState    string
	authVerifier string
	exchangeCode string
	exchangePKCE string
	savedToken   *oauth2.Token
	sentMIME     []byte
}

func (f *fakeGmailOAuthService) Enabled() bool {
	return f.enabled
}

func (f *fakeGmailOAuthService) AuthCodeURL(state, verifier string) string {
	f.authState = state
	f.authVerifier = verifier
	values := url.Values{}
	values.Set("state", state)
	return "https://accounts.google.com/o/oauth2/v2/auth?" + values.Encode()
}

func (f *fakeGmailOAuthService) Exchange(_ context.Context, code, verifier string) (*oauth2.Token, error) {
	f.exchangeCode = code
	f.exchangePKCE = verifier
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	return &oauth2.Token{RefreshToken: "test-refresh-token"}, nil
}

func (f *fakeGmailOAuthService) SaveToken(_ context.Context, token *oauth2.Token) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.savedToken = token
	return nil
}

func (f *fakeGmailOAuthService) Authorized(context.Context) (bool, error) {
	if f.authorizeErr != nil {
		return false, f.authorizeErr
	}
	return f.authorized, nil
}

func (f *fakeGmailOAuthService) Send(_ context.Context, rawMIME []byte) error {
	f.sentMIME = append([]byte(nil), rawMIME...)
	return nil
}

func createGmailTestSession(t *testing.T, env *authTestEnv, userID int64) string {
	t.Helper()
	token := "gmail-test-session-" + time.Now().UTC().Format("20060102150405.000000000")
	expiresAt := time.Now().UTC().Add(time.Hour)
	session := models.Session{
		UserID:       userID,
		TokenDigest:  env.security.TokenDigest(token),
		LastActiveAt: time.Now().UTC(),
		ExpiresAt:    &expiresAt,
	}
	if err := env.db.DB.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	return token
}

func gmailOAuthCallbackRequest(t *testing.T, handler http.Handler, state string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/oauth/gmail/callback?code=test-code&state="+url.QueryEscape(state), nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func gmailOAuthRedirectQuery(t *testing.T, recorder *httptest.ResponseRecorder) url.Values {
	t.Helper()
	if recorder.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want 302; body = %s", recorder.Code, recorder.Body.String())
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse callback location: %v", err)
	}
	return location.Query()
}

func TestGmailOAuthAuthorizationIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	env.server.Config.FrontendURL = "https://edwinslab.test"
	user, _ := createAuthTestAdmin(t, env)
	token := createGmailTestSession(t, env, user.ID)
	service := &fakeGmailOAuthService{enabled: true}
	env.server.GmailAPI = service

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/oauth/gmail/start", token, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("start status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("start response must be non-cacheable, got %q", recorder.Header().Get("Cache-Control"))
	}
	result := authTestDecode[struct {
		Success          bool   `json:"success"`
		AuthorizationURL string `json:"authorization_url"`
	}](t, recorder)
	if !result.Success || result.AuthorizationURL == "" {
		t.Fatalf("unexpected start response: %s", recorder.Body.String())
	}
	authorizationURL, err := url.Parse(result.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	state := authorizationURL.Query().Get("state")
	if state == "" || state != service.authState {
		t.Fatalf("authorization state = %q, fake state = %q", state, service.authState)
	}
	if service.authVerifier == "" {
		t.Fatal("start did not generate a PKCE verifier")
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("start did not set the OAuth state cookie")
	}
	cookie := cookies[0]
	if cookie.Name != gmailOAuthStateCookie || cookie.Path != gmailOAuthStatePath ||
		!cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected state cookie: %+v", cookie)
	}
	cookieState, cookieVerifier, ok := parseGmailOAuthStateCookie(cookie.Value)
	if !ok || cookieState != state || cookieVerifier != service.authVerifier {
		t.Fatalf("state cookie = %q, verifier = %q", cookie.Value, service.authVerifier)
	}

	recorder = gmailOAuthCallbackRequest(t, env.router, state, cookie)
	query := gmailOAuthRedirectQuery(t, recorder)
	if query.Get("gmail") != "connected" {
		t.Fatalf("callback query = %s", query.Encode())
	}
	if !strings.HasPrefix(recorder.Header().Get("Location"), "https://edwinslab.test/app/dashboard") {
		t.Fatalf("callback location = %q", recorder.Header().Get("Location"))
	}
	if service.exchangeCode != "test-code" || service.exchangePKCE != service.authVerifier {
		t.Fatalf("exchange code = %q, verifier = %q", service.exchangeCode, service.exchangePKCE)
	}
	if service.savedToken == nil || service.savedToken.RefreshToken != "test-refresh-token" {
		t.Fatalf("saved token = %+v", service.savedToken)
	}
}

func TestGmailOAuthStartRequiresPrimaryAdmin(t *testing.T) {
	env := newAuthTestEnv(t)
	user, admin := createAuthTestAdmin(t, env)
	service := &fakeGmailOAuthService{enabled: true}
	env.server.GmailAPI = service

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/oauth/gmail/start", "", nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated start status = %d, want 401; body = %s", recorder.Code, recorder.Body.String())
	}

	if err := env.db.DB.Model(&models.Admin{}).Where("id = ?", admin.ID).Update("is_primary", false).Error; err != nil {
		t.Fatalf("demote admin: %v", err)
	}
	token := createGmailTestSession(t, env, user.ID)
	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/oauth/gmail/start", token, nil)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("non-primary start status = %d, want 403; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestGmailOAuthStartDisabled(t *testing.T) {
	env := newAuthTestEnv(t)
	user, _ := createAuthTestAdmin(t, env)
	token := createGmailTestSession(t, env, user.ID)
	env.server.GmailAPI = &fakeGmailOAuthService{}

	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/oauth/gmail/start", token, nil)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled start status = %d, want 503; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestGmailOAuthCallbackRejectsInvalidState(t *testing.T) {
	env := newAuthTestEnv(t)
	env.server.Config.FrontendURL = "https://edwinslab.test"
	env.server.GmailAPI = &fakeGmailOAuthService{enabled: true}

	recorder := gmailOAuthCallbackRequest(t, env.router, "state", nil)
	query := gmailOAuthRedirectQuery(t, recorder)
	if query.Get("gmail_error") != gmailOAuthErrorInvalidState {
		t.Fatalf("gmail_error = %q, want %q", query.Get("gmail_error"), gmailOAuthErrorInvalidState)
	}
}

func TestGmailOAuthCallbackHandlesAccessDenied(t *testing.T) {
	env := newAuthTestEnv(t)
	env.server.Config.FrontendURL = "https://edwinslab.test"
	env.server.GmailAPI = &fakeGmailOAuthService{enabled: true}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/oauth/gmail/callback?error=access_denied", nil)
	recorder := httptest.NewRecorder()
	env.router.ServeHTTP(recorder, request)
	query := gmailOAuthRedirectQuery(t, recorder)
	if query.Get("gmail_error") != gmailOAuthErrorAccessDenied {
		t.Fatalf("gmail_error = %q, want %q", query.Get("gmail_error"), gmailOAuthErrorAccessDenied)
	}
}

func TestGmailOAuthCallbackHandlesStorageFailure(t *testing.T) {
	env := newAuthTestEnv(t)
	env.server.Config.FrontendURL = "https://edwinslab.test"
	env.server.GmailAPI = &fakeGmailOAuthService{enabled: true, saveErr: errors.New("storage failed")}

	cookie := &http.Cookie{Name: gmailOAuthStateCookie, Value: "state.verifier"}
	recorder := gmailOAuthCallbackRequest(t, env.router, "state", cookie)
	query := gmailOAuthRedirectQuery(t, recorder)
	if query.Get("gmail_error") != gmailOAuthErrorStorage {
		t.Fatalf("gmail_error = %q, want %q", query.Get("gmail_error"), gmailOAuthErrorStorage)
	}
}

func TestEmailConfiguredRequiresGmailAuthorization(t *testing.T) {
	env := newAuthTestEnv(t)
	service := &fakeGmailOAuthService{enabled: true, authorized: false}
	env.server.GmailAPI = service
	env.server.Mailer.Provider = mailer.ProviderGmailAPI

	if env.server.EmailConfigured(context.Background()) {
		t.Fatal("EmailConfigured = true without Gmail authorization")
	}
	service.authorized = true
	if !env.server.EmailConfigured(context.Background()) {
		t.Fatal("EmailConfigured = false after Gmail authorization")
	}
	service.authorizeErr = errors.New("credential decryption failed")
	if env.server.EmailConfigured(context.Background()) {
		t.Fatal("EmailConfigured = true when credential check failed")
	}
}

func TestGmailCredentialPersistenceIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32))
	service := gmailapi.New(env.db.DB, config.Config{
		GmailOAuthClientID:      "gmail-test-client",
		GmailOAuthClientSecret:  "gmail-test-secret",
		GmailOAuthRedirectURL:   "https://example.test/api/v1/oauth/gmail/callback",
		GmailTokenEncryptionKey: key,
	})
	if err := service.SaveToken(context.Background(), &oauth2.Token{RefreshToken: "database-refresh-token"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	authorized, err := service.Authorized(context.Background())
	if err != nil {
		t.Fatalf("Authorized: %v", err)
	}
	if !authorized {
		t.Fatal("Authorized = false after SaveToken")
	}
	var credential models.GmailOAuthCredential
	if err := env.db.DB.First(&credential, 1).Error; err != nil {
		t.Fatalf("load credential: %v", err)
	}
	if strings.Contains(credential.EncryptedRefreshToken, "database-refresh-token") {
		t.Fatal("database contains a plaintext refresh token")
	}
}
