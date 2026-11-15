package gmailapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const (
	GmailSendScope      = "https://www.googleapis.com/auth/gmail.send"
	defaultSendEndpoint = "https://gmail.googleapis.com/gmail/v1/users/me/messages/send"
)

var (
	ErrNotConfigured  = errors.New("Gmail API is not configured")
	ErrNotAuthorized  = errors.New("Gmail API has not been authorized")
	ErrNoRefreshToken = errors.New("Gmail OAuth response did not include a refresh token")
)

type Service struct {
	store         credentialStore
	oauthConfig   *oauth2.Config
	httpClient    *http.Client
	sendEndpoint  string
	encryptionKey []byte
	enabled       bool
}

func New(db *gorm.DB, cfg config.Config) *Service {
	var store credentialStore
	if db != nil {
		store = &gormCredentialStore{db: db}
	}
	return newService(cfg, store)
}

func newService(cfg config.Config, store credentialStore) *Service {
	key, keyErr := parseEncryptionKey(cfg.GmailTokenEncryptionKey)
	clientID := strings.TrimSpace(cfg.GmailOAuthClientID)
	clientSecret := strings.TrimSpace(cfg.GmailOAuthClientSecret)
	redirectURL := strings.TrimSpace(cfg.GmailOAuthRedirectURL)
	service := &Service{
		store:         store,
		httpClient:    &http.Client{Timeout: 20 * time.Second},
		sendEndpoint:  defaultSendEndpoint,
		encryptionKey: key,
		enabled:       keyErr == nil && store != nil && clientID != "" && clientSecret != "" && redirectURL != "",
	}
	if service.enabled {
		service.oauthConfig = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{GmailSendScope},
			Endpoint:     google.Endpoint,
		}
	}
	return service
}

func (s *Service) Enabled() bool {
	return s != nil && s.enabled && s.oauthConfig != nil
}

func (s *Service) AuthCodeURL(state, verifier string) string {
	if !s.Enabled() {
		return ""
	}
	return s.oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

func (s *Service) Exchange(ctx context.Context, code, verifier string) (*oauth2.Token, error) {
	if !s.Enabled() {
		return nil, ErrNotConfigured
	}
	return s.oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
}

func (s *Service) SaveToken(ctx context.Context, token *oauth2.Token) error {
	if !s.Enabled() {
		return ErrNotConfigured
	}
	if token == nil {
		return fmt.Errorf("Gmail OAuth token is required")
	}
	refreshToken := strings.TrimSpace(token.RefreshToken)
	if refreshToken == "" {
		existing, err := s.store.Load(ctx)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoRefreshToken
			}
			return err
		}
		decrypted, err := decryptToken(s.encryptionKey, existing.EncryptedRefreshToken)
		if err != nil {
			return err
		}
		refreshToken = string(decrypted)
	}
	if refreshToken == "" {
		return ErrNoRefreshToken
	}
	encrypted, err := encryptToken(s.encryptionKey, []byte(refreshToken))
	if err != nil {
		return fmt.Errorf("encrypt Gmail refresh token: %w", err)
	}
	scopes := GmailSendScope
	if value, ok := token.Extra("scope").(string); ok && strings.TrimSpace(value) != "" {
		scopes = strings.TrimSpace(value)
	}
	tokenType := strings.TrimSpace(token.TokenType)
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return s.store.Save(ctx, &models.GmailOAuthCredential{
		EncryptedRefreshToken: encrypted,
		Scopes:                scopes,
		TokenType:             tokenType,
		AuthorizedAt:          time.Now().UTC(),
	})
}

func (s *Service) Authorized(ctx context.Context) (bool, error) {
	if !s.Enabled() {
		return false, nil
	}
	credential, err := s.store.Load(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	refreshToken, err := decryptToken(s.encryptionKey, credential.EncryptedRefreshToken)
	if err != nil {
		return false, err
	}
	return len(refreshToken) > 0, nil
}

type sendRequest struct {
	Raw string `json:"raw"`
}

func (s *Service) Send(ctx context.Context, rawMIME []byte) error {
	if !s.Enabled() {
		return ErrNotConfigured
	}
	if len(rawMIME) == 0 {
		return fmt.Errorf("Gmail message content is required")
	}
	credential, err := s.store.Load(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotAuthorized
		}
		return err
	}
	refreshToken, err := decryptToken(s.encryptionKey, credential.EncryptedRefreshToken)
	if err != nil {
		return err
	}
	if len(refreshToken) == 0 {
		return ErrNotAuthorized
	}

	payload, err := json.Marshal(sendRequest{Raw: base64.RawURLEncoding.EncodeToString(rawMIME)})
	if err != nil {
		return fmt.Errorf("encode Gmail API request: %w", err)
	}

	tokenCtx := context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)
	client := s.oauthConfig.Client(tokenCtx, &oauth2.Token{RefreshToken: string(refreshToken)})
	client.Timeout = s.httpClient.Timeout

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.sendEndpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create Gmail API request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send email via Gmail API: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("Gmail API returned %d: %s", response.StatusCode, message)
	}
	return nil
}
