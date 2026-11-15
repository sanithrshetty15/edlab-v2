package gmailapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

type memoryCredentialStore struct {
	mu         sync.Mutex
	credential *models.GmailOAuthCredential
	err        error
}

func (s *memoryCredentialStore) Load(context.Context) (*models.GmailOAuthCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	if s.credential == nil {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *s.credential
	return &copy, nil
}

func (s *memoryCredentialStore) Save(_ context.Context, credential *models.GmailOAuthCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := *credential
	s.credential = &copy
	return nil
}

func testGmailConfig(key string) config.Config {
	return config.Config{
		GmailOAuthClientID:      "gmail-test-client",
		GmailOAuthClientSecret:  "gmail-test-secret",
		GmailOAuthRedirectURL:   "https://example.test/api/v1/oauth/gmail/callback",
		GmailTokenEncryptionKey: key,
	}
}

func testGmailService(t *testing.T, store credentialStore) *Service {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(testEncryptionKey())
	service := newService(testGmailConfig(key), store)
	if !service.Enabled() {
		t.Fatal("test Gmail service is not enabled")
	}
	return service
}

func TestAuthCodeURLRequestsOfflineGmailSend(t *testing.T) {
	service := testGmailService(t, &memoryCredentialStore{})
	rawURL := service.AuthCodeURL("state-value", "verifier-value")
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("access_type") != "offline" {
		t.Fatalf("access_type = %q, want offline", query.Get("access_type"))
	}
	if query.Get("prompt") != "consent" {
		t.Fatalf("prompt = %q, want consent", query.Get("prompt"))
	}
	if query.Get("scope") != GmailSendScope {
		t.Fatalf("scope = %q, want %q", query.Get("scope"), GmailSendScope)
	}
	if query.Get("code_challenge") == "" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("auth URL is missing PKCE parameters: %s", rawURL)
	}
	if query.Get("redirect_uri") != "https://example.test/api/v1/oauth/gmail/callback" {
		t.Fatalf("redirect_uri = %q", query.Get("redirect_uri"))
	}
}

func TestSaveTokenEncryptsRefreshToken(t *testing.T) {
	store := &memoryCredentialStore{}
	service := testGmailService(t, store)
	token := (&oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
	}).WithExtra(map[string]any{"scope": GmailSendScope})
	if err := service.SaveToken(context.Background(), token); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	if store.credential == nil {
		t.Fatal("SaveToken did not persist a credential")
	}
	if strings.Contains(store.credential.EncryptedRefreshToken, "refresh-token") {
		t.Fatal("refresh token was stored in plaintext")
	}
	authorized, err := service.Authorized(context.Background())
	if err != nil {
		t.Fatalf("Authorized: %v", err)
	}
	if !authorized {
		t.Fatal("Authorized = false after SaveToken")
	}
}

func TestSaveTokenRequiresRefreshToken(t *testing.T) {
	service := testGmailService(t, &memoryCredentialStore{})
	err := service.SaveToken(context.Background(), &oauth2.Token{AccessToken: "access-token"})
	if !errors.Is(err, ErrNoRefreshToken) {
		t.Fatalf("SaveToken error = %v, want %v", err, ErrNoRefreshToken)
	}
}

func TestSendRefreshesTokenAndPostsRawMIME(t *testing.T) {
	var tokenRequests int
	var sendRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenRequests++
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			if r.Form.Get("grant_type") != "refresh_token" {
				t.Fatalf("grant_type = %q", r.Form.Get("grant_type"))
			}
			if r.Form.Get("refresh_token") != "refresh-token" {
				t.Fatalf("refresh_token = %q", r.Form.Get("refresh_token"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "/send":
			sendRequests++
			if r.Header.Get("Authorization") != "Bearer access-token" {
				t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
			}
			var request sendRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode send request: %v", err)
			}
			raw, err := base64.RawURLEncoding.DecodeString(request.Raw)
			if err != nil {
				t.Fatalf("decode raw MIME: %v", err)
			}
			if string(raw) != "MIME-Version: 1.0\r\n\r\nbody" {
				t.Fatalf("raw MIME = %q", raw)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"message-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &memoryCredentialStore{}
	service := testGmailService(t, store)
	service.httpClient = server.Client()
	service.oauthConfig.Endpoint = oauth2.Endpoint{
		AuthURL:  server.URL + "/authorize",
		TokenURL: server.URL + "/token",
	}
	service.sendEndpoint = server.URL + "/send"
	if err := service.SaveToken(context.Background(), &oauth2.Token{RefreshToken: "refresh-token"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	if err := service.Send(context.Background(), []byte("MIME-Version: 1.0\r\n\r\nbody")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if tokenRequests != 1 {
		t.Fatalf("token requests = %d, want 1", tokenRequests)
	}
	if sendRequests != 1 {
		t.Fatalf("send requests = %d, want 1", sendRequests)
	}
}

func TestSendWithoutCredentialFailsClosed(t *testing.T) {
	service := testGmailService(t, &memoryCredentialStore{})
	err := service.Send(context.Background(), []byte("body"))
	if !errors.Is(err, ErrNotAuthorized) {
		t.Fatalf("Send error = %v, want %v", err, ErrNotAuthorized)
	}
}

func TestServiceDisabledWithoutCompleteConfiguration(t *testing.T) {
	service := newService(config.Config{}, &memoryCredentialStore{})
	if service.Enabled() {
		t.Fatal("Gmail service enabled without complete configuration")
	}
	if _, err := service.Exchange(context.Background(), "code", "verifier"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Exchange error = %v, want %v", err, ErrNotConfigured)
	}
}

func TestSendPropagatesGmailAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}
		http.Error(w, "quota exceeded", http.StatusTooManyRequests)
	}))
	defer server.Close()

	service := testGmailService(t, &memoryCredentialStore{})
	service.httpClient = server.Client()
	service.oauthConfig.Endpoint = oauth2.Endpoint{
		AuthURL:  server.URL + "/authorize",
		TokenURL: server.URL + "/token",
	}
	service.sendEndpoint = server.URL + "/send"
	if err := service.SaveToken(context.Background(), &oauth2.Token{RefreshToken: "refresh-token"}); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	err := service.Send(context.Background(), []byte("body"))
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("Send error = %v, want HTTP 429", err)
	}
}
