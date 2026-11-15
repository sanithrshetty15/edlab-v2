package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

const (
	gmailOAuthStateCookie = "edwinslab_gmail_oauth_state"
	gmailOAuthStateTTL    = 10 * time.Minute
	gmailOAuthStatePath   = "/api/v1/oauth/gmail"

	gmailOAuthErrorNotConfigured = "not_configured"
	gmailOAuthErrorAccessDenied  = "access_denied"
	gmailOAuthErrorInvalidState  = "invalid_state"
	gmailOAuthErrorExchange      = "exchange_failed"
	gmailOAuthErrorStorage       = "storage_failed"
)

type gmailOAuthService interface {
	Enabled() bool
	AuthCodeURL(state, verifier string) string
	Exchange(ctx context.Context, code, verifier string) (*oauth2.Token, error)
	SaveToken(ctx context.Context, token *oauth2.Token) error
	Authorized(ctx context.Context) (bool, error)
	Send(ctx context.Context, rawMIME []byte) error
}

func (s *Server) registerGmailOAuthRoutes(api *gin.RouterGroup) {
	gmail := api.Group("/oauth/gmail")
	gmail.Use(noStore)
	gmail.POST("/start", s.RateLimit("gmail-oauth-start", 10, 5*time.Minute), s.RequireAuth(), s.RequireAdmin(), s.RequirePrimaryAdmin(), s.gmailOAuthStart)
	gmail.GET("/callback", s.gmailOAuthCallback)
}

func (s *Server) gmailOAuthStart(c *gin.Context) {
	if s.GmailAPI == nil || !s.GmailAPI.Enabled() {
		fail(c, http.StatusServiceUnavailable, "Gmail API OAuth is not configured")
		return
	}
	state, err := randomURLToken(32)
	if err != nil {
		s.Log.Error("gmail oauth state generation failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to start Gmail authorization")
		return
	}
	verifier, err := randomURLToken(32)
	if err != nil {
		s.Log.Error("gmail oauth PKCE verifier generation failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to start Gmail authorization")
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     gmailOAuthStateCookie,
		Value:    state + "." + verifier,
		Path:     gmailOAuthStatePath,
		MaxAge:   int(gmailOAuthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.Config.AppEnv == "production",
		SameSite: http.SameSiteLaxMode,
	})

	authorizationURL := s.GmailAPI.AuthCodeURL(state, verifier)
	if authorizationURL == "" {
		fail(c, http.StatusServiceUnavailable, "Gmail API OAuth is not configured")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"authorization_url": authorizationURL,
	})
}

func (s *Server) gmailOAuthCallback(c *gin.Context) {
	s.clearGmailOAuthStateCookie(c)
	if s.GmailAPI == nil || !s.GmailAPI.Enabled() {
		s.redirectGmailOAuthError(c, gmailOAuthErrorNotConfigured)
		return
	}
	if c.Query("error") != "" {
		s.redirectGmailOAuthError(c, gmailOAuthErrorAccessDenied)
		return
	}

	state := strings.TrimSpace(c.Query("state"))
	cookieValue, err := c.Cookie(gmailOAuthStateCookie)
	if err != nil || state == "" || cookieValue == "" {
		s.redirectGmailOAuthError(c, gmailOAuthErrorInvalidState)
		return
	}
	cookieState, verifier, ok := parseGmailOAuthStateCookie(cookieValue)
	if !ok || subtle.ConstantTimeCompare([]byte(state), []byte(cookieState)) != 1 {
		s.redirectGmailOAuthError(c, gmailOAuthErrorInvalidState)
		return
	}

	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		s.redirectGmailOAuthError(c, gmailOAuthErrorExchange)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	token, err := s.GmailAPI.Exchange(ctx, code, verifier)
	if err != nil {
		s.Log.Warn("gmail oauth code exchange failed", "error", err)
		s.redirectGmailOAuthError(c, gmailOAuthErrorExchange)
		return
	}
	if err := s.GmailAPI.SaveToken(ctx, token); err != nil {
		s.Log.Error("gmail oauth credential storage failed", "error", err)
		s.redirectGmailOAuthError(c, gmailOAuthErrorStorage)
		return
	}
	s.Log.Info("gmail oauth authorization completed")
	s.redirectGmailOAuthSuccess(c)
}

func parseGmailOAuthStateCookie(value string) (string, string, bool) {
	state, verifier, found := strings.Cut(value, ".")
	if !found || state == "" || verifier == "" {
		return "", "", false
	}
	return state, verifier, true
}

func (s *Server) clearGmailOAuthStateCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     gmailOAuthStateCookie,
		Value:    "",
		Path:     gmailOAuthStatePath,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   s.Config.AppEnv == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) redirectGmailOAuthSuccess(c *gin.Context) {
	values := url.Values{}
	values.Set("gmail", "connected")
	c.Redirect(http.StatusFound, s.gmailOAuthFrontendURL(values))
}

func (s *Server) redirectGmailOAuthError(c *gin.Context, code string) {
	values := url.Values{}
	values.Set("gmail_error", code)
	c.Redirect(http.StatusFound, s.gmailOAuthFrontendURL(values))
}

func (s *Server) gmailOAuthFrontendURL(values url.Values) string {
	base := strings.TrimRight(strings.TrimSpace(s.Config.FrontendURL), "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	target := base + "/app/dashboard"
	if encoded := values.Encode(); encoded != "" {
		target += "?" + encoded
	}
	return target
}
