package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const (
	googleLoginStateCookie = "edwinslab_google_login_state"
	googleLoginStateTTL    = 10 * time.Minute
	googleLoginHandoffTTL  = 2 * time.Minute
	googleLoginStatePath   = "/api/v1/oauth/google"

	googleIssuer      = "https://accounts.google.com"
	googleIssuerShort = "accounts.google.com"

	googleLoginErrorNotConfigured = "not_configured"
	googleLoginErrorAccessDenied  = "access_denied"
	googleLoginErrorInvalidState  = "invalid_state"
	googleLoginErrorExchange      = "exchange_failed"
	googleLoginErrorInvalidToken  = "invalid_token"
	googleLoginErrorNotAuthorized = "not_authorized"
)

var (
	errGoogleLoginNotAuthorized  = errors.New("google account is not an authorized administrator")
	errGoogleLoginHandoffInvalid = errors.New("google login handoff is invalid or expired")
)

type googleLoginService struct {
	enabled         bool
	clientID        string
	config          *oauth2.Config
	exchange        func(context.Context, string, string) (*oauth2.Token, error)
	validateIDToken func(context.Context, string, string) (*idtoken.Payload, error)
}

func newGoogleLoginService(cfg config.Config) *googleLoginService {
	clientID := strings.TrimSpace(cfg.GoogleLoginClientID)
	clientSecret := strings.TrimSpace(cfg.GoogleLoginSecret)
	redirectURL := strings.TrimSpace(cfg.GoogleLoginRedirectURL)
	service := &googleLoginService{
		enabled:  clientID != "" && clientSecret != "" && redirectURL != "",
		clientID: clientID,
	}
	if !service.enabled {
		return service
	}

	service.config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	service.exchange = func(ctx context.Context, code, verifier string) (*oauth2.Token, error) {
		return service.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	}
	service.validateIDToken = idtoken.Validate
	return service
}

func (g *googleLoginService) authCodeURL(state, verifier string) string {
	return g.config.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
}

func (s *Server) registerGoogleLoginRoutes(api *gin.RouterGroup) {
	googleLogin := api.Group("/oauth/google")
	googleLogin.Use(noStore)
	googleLogin.GET("/start", s.RateLimit("google-login-start", 20, 5*time.Minute), s.googleLoginStart)
	googleLogin.GET("/callback", s.googleLoginCallback)
	googleLogin.POST("/exchange", s.RateLimit("google-login-exchange", 20, 5*time.Minute), s.googleLoginExchange)
}

func (s *Server) googleLoginStart(c *gin.Context) {
	if s.GoogleLogin == nil || !s.GoogleLogin.enabled {
		s.redirectGoogleLoginError(c, googleLoginErrorNotConfigured)
		return
	}

	state, err := randomURLToken(32)
	if err != nil {
		s.Log.Error("google login state generation failed", "error", err)
		s.redirectGoogleLoginError(c, googleLoginErrorExchange)
		return
	}
	verifier, err := randomURLToken(32)
	if err != nil {
		s.Log.Error("google login PKCE verifier generation failed", "error", err)
		s.redirectGoogleLoginError(c, googleLoginErrorExchange)
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     googleLoginStateCookie,
		Value:    state + "." + verifier,
		Path:     googleLoginStatePath,
		MaxAge:   int(googleLoginStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.Config.AppEnv == "production",
		SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusFound, s.GoogleLogin.authCodeURL(state, verifier))
}

func (s *Server) googleLoginCallback(c *gin.Context) {
	s.clearGoogleLoginStateCookie(c)
	if s.GoogleLogin == nil || !s.GoogleLogin.enabled {
		s.redirectGoogleLoginError(c, googleLoginErrorNotConfigured)
		return
	}
	if c.Query("error") != "" {
		s.redirectGoogleLoginError(c, googleLoginErrorAccessDenied)
		return
	}

	state := strings.TrimSpace(c.Query("state"))
	cookieValue, err := c.Cookie(googleLoginStateCookie)
	if err != nil || state == "" || cookieValue == "" {
		s.redirectGoogleLoginError(c, googleLoginErrorInvalidState)
		return
	}
	cookieState, verifier, ok := parseGoogleLoginStateCookie(cookieValue)
	if !ok || subtle.ConstantTimeCompare([]byte(state), []byte(cookieState)) != 1 {
		s.redirectGoogleLoginError(c, googleLoginErrorInvalidState)
		return
	}

	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		s.redirectGoogleLoginError(c, googleLoginErrorExchange)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	token, err := s.GoogleLogin.exchange(ctx, code, verifier)
	if err != nil {
		s.Log.Warn("google login code exchange failed", "error", err)
		s.redirectGoogleLoginError(c, googleLoginErrorExchange)
		return
	}
	rawIDToken, _ := token.Extra("id_token").(string)
	if strings.TrimSpace(rawIDToken) == "" {
		s.Log.Warn("google login response did not include an id token")
		s.redirectGoogleLoginError(c, googleLoginErrorInvalidToken)
		return
	}

	payload, err := s.GoogleLogin.validateIDToken(ctx, rawIDToken, s.GoogleLogin.clientID)
	if err != nil {
		s.Log.Warn("google login id token validation failed", "error", err)
		s.redirectGoogleLoginError(c, googleLoginErrorInvalidToken)
		return
	}
	email, err := googleVerifiedEmail(payload, s.GoogleLogin.clientID)
	if err != nil {
		s.Log.Warn("google login id token claims rejected", "error", err)
		s.redirectGoogleLoginError(c, googleLoginErrorInvalidToken)
		return
	}

	user, admin, err := s.findGoogleAdmin(ctx, email)
	if err != nil {
		if !errors.Is(err, errGoogleLoginNotAuthorized) {
			s.Log.Error("google admin lookup failed", "error", err)
			s.redirectGoogleLoginError(c, googleLoginErrorExchange)
			return
		}
		s.Log.Warn("google login rejected for unauthorized account")
		s.redirectGoogleLoginError(c, googleLoginErrorNotAuthorized)
		return
	}

	handoffCode, err := s.createGoogleLoginHandoff(ctx, user.ID)
	if err != nil {
		s.Log.Error("google login handoff creation failed", "user_id", user.ID, "error", err)
		s.redirectGoogleLoginError(c, googleLoginErrorExchange)
		return
	}
	s.Log.Info("google admin login verified", "user_id", user.ID, "admin_id", admin.ID, "email", maskEmail(user.Email))
	s.redirectGoogleLoginSuccess(c, handoffCode)
}

func (s *Server) googleLoginExchange(c *gin.Context) {
	var request struct {
		Code string `json:"code"`
	}
	if !bindJSON(c, &request) {
		return
	}
	code := strings.TrimSpace(request.Code)
	if code == "" {
		fail(c, http.StatusUnprocessableEntity, "Google login code is required")
		return
	}

	ctx := c.Request.Context()
	handoff, err := s.consumeGoogleLoginHandoff(ctx, code)
	if err != nil {
		if !errors.Is(err, errGoogleLoginHandoffInvalid) {
			s.Log.Error("google login handoff consumption failed", "error", err)
			fail(c, http.StatusInternalServerError, "Unable to complete Google sign-in")
			return
		}
		fail(c, http.StatusUnauthorized, "Google sign-in has expired. Please try again.")
		return
	}

	var user models.User
	if err := s.DB.WithContext(ctx).First(&user, handoff.UserID).Error; err != nil {
		s.Log.Error("google login user lookup failed", "user_id", handoff.UserID, "error", err)
		fail(c, http.StatusUnauthorized, "Google account is not authorized")
		return
	}
	if user.Role != "admin" || !user.Active {
		fail(c, http.StatusForbidden, "Administrator access is required")
		return
	}
	var admin models.Admin
	if err := s.DB.WithContext(ctx).Where("user_id = ?", user.ID).First(&admin).Error; err != nil {
		fail(c, http.StatusForbidden, "Administrator access is required")
		return
	}

	s.respondAdminSession(c, &user, &admin, gin.H{"provider": "google"})
}

func (s *Server) findGoogleAdmin(ctx context.Context, email string) (*models.User, *models.Admin, error) {
	var user models.User
	err := s.DB.WithContext(ctx).
		Where("LOWER(email) = ? AND role = ? AND active = ?", email, "admin", true).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, errGoogleLoginNotAuthorized
	}
	if err != nil {
		return nil, nil, err
	}

	var admin models.Admin
	if err := s.DB.WithContext(ctx).Where("user_id = ?", user.ID).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errGoogleLoginNotAuthorized
		}
		return nil, nil, err
	}
	return &user, &admin, nil
}

func (s *Server) createGoogleLoginHandoff(ctx context.Context, userID int64) (string, error) {
	code, err := randomURLToken(32)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	record := models.OAuthLoginHandoff{
		CodeDigest: s.Security.TokenDigest(code),
		UserID:     userID,
		ExpiresAt:  now.Add(googleLoginHandoffTTL),
	}
	if err := s.DB.WithContext(ctx).Create(&record).Error; err != nil {
		return "", err
	}
	_ = s.DB.WithContext(ctx).
		Where("expires_at < ?", now.Add(-time.Hour)).
		Delete(&models.OAuthLoginHandoff{}).Error
	return code, nil
}

func (s *Server) consumeGoogleLoginHandoff(ctx context.Context, code string) (*models.OAuthLoginHandoff, error) {
	var handoff models.OAuthLoginHandoff
	digest := s.Security.TokenDigest(code)
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code_digest = ?", digest).
			First(&handoff).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errGoogleLoginHandoffInvalid
		}
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if handoff.UsedAt != nil || !handoff.ExpiresAt.After(now) {
			return errGoogleLoginHandoffInvalid
		}
		handoff.UsedAt = &now
		return tx.Model(&handoff).Update("used_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return &handoff, nil
}

func googleVerifiedEmail(payload *idtoken.Payload, clientID string) (string, error) {
	if payload == nil {
		return "", errors.New("google id token payload is missing")
	}
	if payload.Issuer != googleIssuer && payload.Issuer != googleIssuerShort {
		return "", fmt.Errorf("unexpected google id token issuer %q", payload.Issuer)
	}
	if payload.Audience != clientID {
		return "", fmt.Errorf("unexpected google id token audience %q", payload.Audience)
	}
	if payload.Expires <= time.Now().UTC().Unix() {
		return "", errors.New("google id token is expired")
	}
	email, _ := payload.Claims["email"].(string)
	email = strings.ToLower(strings.TrimSpace(email))
	if !emailPattern.MatchString(email) {
		return "", errors.New("google id token does not contain a valid email")
	}
	verified, _ := payload.Claims["email_verified"].(bool)
	if !verified {
		return "", errors.New("google email is not verified")
	}
	return email, nil
}

func randomURLToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func parseGoogleLoginStateCookie(value string) (string, string, bool) {
	state, verifier, found := strings.Cut(value, ".")
	if !found || state == "" || verifier == "" {
		return "", "", false
	}
	return state, verifier, true
}

func (s *Server) clearGoogleLoginStateCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     googleLoginStateCookie,
		Value:    "",
		Path:     googleLoginStatePath,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   s.Config.AppEnv == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) redirectGoogleLoginSuccess(c *gin.Context, code string) {
	values := url.Values{}
	values.Set("google_code", code)
	c.Redirect(http.StatusFound, s.frontendLoginURL(values))
}

func (s *Server) redirectGoogleLoginError(c *gin.Context, code string) {
	values := url.Values{}
	values.Set("google_error", code)
	c.Redirect(http.StatusFound, s.frontendLoginURL(values))
}

func (s *Server) frontendLoginURL(values url.Values) string {
	base := strings.TrimRight(strings.TrimSpace(s.Config.FrontendURL), "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	target := base + "/login"
	if encoded := values.Encode(); encoded != "" {
		target += "?" + encoded
	}
	return target
}
