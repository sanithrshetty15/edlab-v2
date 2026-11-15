package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/httpx"
	"github.com/sanithrshetty15/edwinslab/backend/internal/jobs"
	"github.com/sanithrshetty15/edwinslab/backend/internal/mailer"
	"github.com/sanithrshetty15/edwinslab/backend/internal/ratelimit"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

type Server struct {
	DB          *gorm.DB
	Config      config.Config
	Log         *slog.Logger
	Security    *security.Manager
	Mailer      *mailer.Mailer
	Jobs        *jobs.Manager
	Limiter     *ratelimit.Limiter
	GoogleLogin *googleLoginService
}

func New(db *gorm.DB, cfg config.Config, log *slog.Logger, mail *mailer.Mailer, jobManager *jobs.Manager) *Server {
	server := &Server{
		DB:          db,
		Config:      cfg,
		Log:         log,
		Security:    security.New(cfg.JWTSecret, cfg.SessionSecret),
		Mailer:      mail,
		Jobs:        jobManager,
		Limiter:     ratelimit.New(),
		GoogleLogin: newGoogleLoginService(cfg),
	}
	server.registerJobHandlers()
	return server
}

func (s *Server) Router() *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.HandleMethodNotAllowed = true
	router.Use(s.recoveryMiddleware(), s.requestLogger(), s.securityHeaders(), s.corsMiddleware())

	router.GET("/health", s.health)
	router.GET("/up", s.health)

	api := router.Group("/api/v1")
	api.GET("/health", s.health)

	s.registerAuthRoutes(api)
	s.registerGoogleLoginRoutes(api)
	s.registerDeviceRoutes(api)
	s.registerMemberRoutes(api)
	s.registerAdminRoutes(api)
	s.registerMemberManagementRoutes(api)
	s.registerAttendanceRoutes(api)
	s.registerBiometricRoutes(api)
	s.registerDoorAccessRoutes(api)
	s.registerMessageRoutes(api)
	s.registerNotificationRoutes(api)
	s.registerAnnouncementRoutes(api)
	s.registerEventRoutes(api)
	s.registerTaskRoutes(api)
	s.registerInventoryRoutes(api)
	s.registerBorrowingRoutes(api)
	s.registerReturnRoutes(api)
	s.registerContactRoutes(api)
	s.registerFAQRoutes(api)
	s.registerAnalyticsRoutes(api)
	s.registerAuditLogRoutes(api)

	router.NoRoute(func(c *gin.Context) {
		httpx.Error(c, http.StatusNotFound, "Route not found")
	})
	router.NoMethod(func(c *gin.Context) {
		httpx.Error(c, http.StatusMethodNotAllowed, "Method not allowed")
	})
	return router
}

func (s *Server) health(c *gin.Context) {
	sqlDB, err := s.DB.DB()
	if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":      "error",
			"application": "Edwins Lab API",
			"version":     "v1",
			"database":    "unavailable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"application": "Edwins Lab API",
		"version":     "v1",
		"database":    "ok",
	})
}

func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		s.Log.Info("http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
		)
	}
}

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		s.Log.Error("panic recovered", "path", c.Request.URL.Path, "error", recovered)
		if !c.Writer.Written() {
			httpx.Error(c, http.StatusInternalServerError, "An unexpected error occurred")
		}
	})
}

func (s *Server) securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if s.Config.AppEnv == "production" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(s.Config.CORSAllowedOrigins))
	for _, origin := range s.Config.CORSAllowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			_, ok := allowed[origin]
			if ok || contains(allowed, "*") {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Idempotency-Key, X-Device-Api-Key, X-Device-Signature")
				c.Header("Access-Control-Max-Age", "600")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func contains(values map[string]struct{}, key string) bool {
	_, ok := values[key]
	return ok
}
