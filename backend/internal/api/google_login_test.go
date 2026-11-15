package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

const testGoogleTransitionCode = "test-google-transition-code"

type googleExchangeResponse struct {
	Success   bool       `json:"success"`
	Token     string     `json:"token"`
	TokenType string     `json:"token_type"`
	ExpiresAt *time.Time `json:"expires_at"`
	Admin     struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		IsPrimary bool   `json:"is_primary"`
	} `json:"admin"`
}

func googleTestPayload(email string) *idtoken.Payload {
	return &idtoken.Payload{
		Issuer:   googleIssuer,
		Audience: "google-login-test-client",
		Expires:  time.Now().UTC().Add(5 * time.Minute).Unix(),
		Claims: map[string]any{
			"email":          email,
			"email_verified": true,
		},
	}
}

func configureTestGoogleLogin(env *authTestEnv, payload *idtoken.Payload) {
	env.server.GoogleLogin = &googleLoginService{
		enabled:  true,
		clientID: "google-login-test-client",
		config: &oauth2.Config{
			ClientID:     "google-login-test-client",
			ClientSecret: "google-login-test-secret",
			RedirectURL:  "https://example.test/api/v1/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
		exchange: func(context.Context, string, string) (*oauth2.Token, error) {
			return (&oauth2.Token{AccessToken: "test-access-token"}).WithExtra(map[string]any{
				"id_token": "test-id-token",
			}), nil
		},
		validateIDToken: func(context.Context, string, string) (*idtoken.Payload, error) {
			return payload, nil
		},
	}
}

func googleCallbackRequest(t *testing.T, handler http.Handler, state, cookieValue string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/oauth/google/callback?code="+url.QueryEscape(testGoogleTransitionCode)+"&state="+url.QueryEscape(state), nil)
	if cookieValue != "" {
		request.AddCookie(&http.Cookie{Name: googleLoginStateCookie, Value: cookieValue})
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func googleRedirectQuery(t *testing.T, recorder *httptest.ResponseRecorder) url.Values {
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

func TestGoogleVerifiedEmail(t *testing.T) {
	tests := []struct {
		name    string
		payload *idtoken.Payload
		want    string
		wantErr bool
	}{
		{name: "valid", payload: googleTestPayload("Admin@Example.com"), want: "admin@example.com"},
		{name: "wrong issuer", payload: func() *idtoken.Payload {
			payload := googleTestPayload("admin@example.com")
			payload.Issuer = "https://issuer.example"
			return payload
		}(), wantErr: true},
		{name: "wrong audience", payload: func() *idtoken.Payload {
			payload := googleTestPayload("admin@example.com")
			payload.Audience = "another-client"
			return payload
		}(), wantErr: true},
		{name: "expired", payload: func() *idtoken.Payload {
			payload := googleTestPayload("admin@example.com")
			payload.Expires = time.Now().UTC().Add(-time.Minute).Unix()
			return payload
		}(), wantErr: true},
		{name: "unverified email", payload: func() *idtoken.Payload {
			payload := googleTestPayload("admin@example.com")
			payload.Claims["email_verified"] = false
			return payload
		}(), wantErr: true},
		{name: "invalid email", payload: googleTestPayload("not-an-email"), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := googleVerifiedEmail(test.payload, "google-login-test-client")
			if test.wantErr {
				if err == nil {
					t.Fatalf("googleVerifiedEmail() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("googleVerifiedEmail() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("googleVerifiedEmail() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGoogleAdminLoginIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	user, admin := createAuthTestAdmin(t, env)
	configureTestGoogleLogin(env, googleTestPayload(strings.ToUpper(user.Email)))

	recorder := authTestRequest(t, env.router, http.MethodGet, "/api/v1/oauth/google/start", "", nil)
	if recorder.Code != http.StatusFound {
		t.Fatalf("start status = %d, want 302; body = %s", recorder.Code, recorder.Body.String())
	}
	setCookie := recorder.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, googleLoginStateCookie+"=") ||
		!strings.Contains(setCookie, "HttpOnly") ||
		!strings.Contains(setCookie, "SameSite=Lax") {
		t.Fatalf("start did not set a secure state cookie: %q", setCookie)
	}
	startLocation, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse start location: %v", err)
	}
	state := startLocation.Query().Get("state")
	if state == "" || startLocation.Query().Get("code_challenge") == "" {
		t.Fatalf("start URL is missing PKCE state: %s", startLocation.String())
	}
	if startLocation.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("start URL is missing S256 PKCE: %s", startLocation.String())
	}
	if redirectURI := startLocation.Query().Get("redirect_uri"); redirectURI != "https://example.test/api/v1/oauth/google/callback" {
		t.Fatalf("start redirect_uri = %q", redirectURI)
	}
	if scope := startLocation.Query().Get("scope"); !strings.Contains(scope, "openid") ||
		!strings.Contains(scope, "email") || !strings.Contains(scope, "profile") {
		t.Fatalf("start scope = %q", scope)
	}

	cookieValue := strings.TrimPrefix(strings.Split(setCookie, ";")[0], googleLoginStateCookie+"=")
	recorder = googleCallbackRequest(t, env.router, state, cookieValue)
	query := googleRedirectQuery(t, recorder)
	if query.Get("google_error") != "" {
		t.Fatalf("unexpected Google login error: %s", query.Get("google_error"))
	}
	transitionCode := query.Get("google_code")
	if transitionCode == "" {
		t.Fatalf("callback did not issue a handoff code: %s", recorder.Header().Get("Location"))
	}

	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/oauth/google/exchange", "", map[string]any{
		"code": transitionCode,
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("exchange status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("exchange response must be non-cacheable, got %q", recorder.Header().Get("Cache-Control"))
	}
	result := authTestDecode[googleExchangeResponse](t, recorder)
	if !result.Success || result.Token == "" || result.TokenType != "Bearer" {
		t.Fatalf("unexpected exchange response: %s", recorder.Body.String())
	}
	if result.Admin.ID != admin.ID || result.Admin.Email != user.Email {
		t.Fatalf("exchange returned wrong admin: %+v", result.Admin)
	}
	if result.ExpiresAt == nil || !result.ExpiresAt.After(time.Now().UTC()) {
		t.Fatalf("exchange returned invalid expiry: %+v", result.ExpiresAt)
	}

	var handoff models.OAuthLoginHandoff
	if err := env.db.DB.Where("user_id = ?", user.ID).First(&handoff).Error; err != nil {
		t.Fatalf("load handoff: %v", err)
	}
	if handoff.UsedAt == nil {
		t.Fatal("handoff was not marked as used")
	}
	if handoff.CodeDigest == transitionCode {
		t.Fatal("raw handoff code was stored instead of a digest")
	}

	recorder = authTestRequest(t, env.router, http.MethodPost, "/api/v1/oauth/google/exchange", "", map[string]any{
		"code": transitionCode,
	})
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("reused handoff status = %d, want 401; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestGoogleLoginRejectsUnregisteredAccountIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	var before int64
	env.db.DB.Model(&models.OAuthLoginHandoff{}).Count(&before)
	configureTestGoogleLogin(env, googleTestPayload("unknown-admin@example.com"))

	recorder := googleCallbackRequest(t, env.router, "state", "state.verifier")
	query := googleRedirectQuery(t, recorder)
	if query.Get("google_error") != googleLoginErrorNotAuthorized {
		t.Fatalf("google_error = %q, want %q", query.Get("google_error"), googleLoginErrorNotAuthorized)
	}
	var after int64
	env.db.DB.Model(&models.OAuthLoginHandoff{}).Count(&after)
	if after != before {
		t.Fatalf("unauthorized Google login created %d handoffs", after-before)
	}
}

func TestGoogleLoginRejectsMemberAccountIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	user, _ := createAuthTestMember(t, env, "CorrectHorseBattery1!")
	configureTestGoogleLogin(env, googleTestPayload(user.Email))

	recorder := googleCallbackRequest(t, env.router, "state", "state.verifier")
	query := googleRedirectQuery(t, recorder)
	if query.Get("google_error") != googleLoginErrorNotAuthorized {
		t.Fatalf("google_error = %q, want %q", query.Get("google_error"), googleLoginErrorNotAuthorized)
	}
}

func TestGoogleLoginRejectsInvalidStateIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	configureTestGoogleLogin(env, googleTestPayload("admin@example.com"))

	recorder := googleCallbackRequest(t, env.router, "state", "")
	query := googleRedirectQuery(t, recorder)
	if query.Get("google_error") != googleLoginErrorInvalidState {
		t.Fatalf("google_error = %q, want %q", query.Get("google_error"), googleLoginErrorInvalidState)
	}
}

func TestGoogleLoginRejectsInvalidIDTokenIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	configureTestGoogleLogin(env, googleTestPayload("admin@example.com"))
	env.server.GoogleLogin.validateIDToken = func(context.Context, string, string) (*idtoken.Payload, error) {
		return nil, errors.New("signature verification failed")
	}

	recorder := googleCallbackRequest(t, env.router, "state", "state.verifier")
	query := googleRedirectQuery(t, recorder)
	if query.Get("google_error") != googleLoginErrorInvalidToken {
		t.Fatalf("google_error = %q, want %q", query.Get("google_error"), googleLoginErrorInvalidToken)
	}
}

func TestGoogleLoginRejectsUnverifiedEmailIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	payload := googleTestPayload("admin@example.com")
	payload.Claims["email_verified"] = false
	configureTestGoogleLogin(env, payload)

	recorder := googleCallbackRequest(t, env.router, "state", "state.verifier")
	query := googleRedirectQuery(t, recorder)
	if query.Get("google_error") != googleLoginErrorInvalidToken {
		t.Fatalf("google_error = %q, want %q", query.Get("google_error"), googleLoginErrorInvalidToken)
	}
}

func TestGoogleLoginExchangeRejectsUnknownHandoffIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/oauth/google/exchange", "", map[string]any{
		"code": "unknown-handoff-code",
	})
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unknown handoff status = %d, want 401; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestGoogleLoginHandoffIsSingleUseIntegration(t *testing.T) {
	env := newAuthTestEnv(t)
	user, _ := createAuthTestAdmin(t, env)
	code, err := env.server.createGoogleLoginHandoff(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("create handoff: %v", err)
	}

	handoff, err := env.server.consumeGoogleLoginHandoff(context.Background(), code)
	if err != nil {
		t.Fatalf("consume handoff: %v", err)
	}
	if handoff.UserID != user.ID {
		t.Fatalf("handoff user = %d, want %d", handoff.UserID, user.ID)
	}
	if _, err := env.server.consumeGoogleLoginHandoff(context.Background(), code); !errors.Is(err, errGoogleLoginHandoffInvalid) {
		t.Fatalf("second consume error = %v, want %v", err, errGoogleLoginHandoffInvalid)
	}
}

func TestGoogleLoginExchangeRejectsMissingCode(t *testing.T) {
	env := newAuthTestEnv(t)
	recorder := authTestRequest(t, env.router, http.MethodPost, "/api/v1/oauth/google/exchange", "", map[string]any{
		"code": "",
	})
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing code status = %d, want 422; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestGoogleLoginDisabledRedirectsToLogin(t *testing.T) {
	env := newAuthTestEnv(t)
	recorder := authTestRequest(t, env.router, http.MethodGet, "/api/v1/oauth/google/start", "", nil)
	query := googleRedirectQuery(t, recorder)
	if query.Get("google_error") != googleLoginErrorNotConfigured {
		t.Fatalf("google_error = %q, want %q", query.Get("google_error"), googleLoginErrorNotConfigured)
	}
}

func TestGoogleLoginServiceDisabledWithoutConfig(t *testing.T) {
	service := newGoogleLoginService(config.Config{})
	if service.enabled {
		t.Fatal("Google login service enabled without credentials")
	}
}
