package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/httpx"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"gorm.io/datatypes"
)

const (
	contextUserKey    = "current_user"
	contextMemberKey  = "current_member"
	contextAdminKey   = "current_admin"
	contextDeviceKey  = "current_device"
	contextSessionKey = "current_session"
)

func (s *Server) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			httpx.Unauthorized(c, "Authentication token is required")
			c.Abort()
			return
		}
		token := strings.TrimSpace(header[7:])
		if token == "" {
			httpx.Unauthorized(c, "Authentication token is required")
			c.Abort()
			return
		}
		var session models.Session
		now := time.Now().UTC()
		if err := s.DB.WithContext(c.Request.Context()).Where("token_digest = ?", s.Security.TokenDigest(token)).First(&session).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				httpx.Unauthorized(c, "Invalid or expired session")
				c.Abort()
				return
			}
			s.Log.Error("session lookup failed", "error", err)
			httpx.Error(c, http.StatusInternalServerError, "Unable to authenticate")
			c.Abort()
			return
		}
		if session.ExpiresAt != nil && !session.ExpiresAt.After(now) {
			httpx.Unauthorized(c, "Invalid or expired session")
			c.Abort()
			return
		}
		var user models.User
		if err := s.DB.WithContext(c.Request.Context()).First(&user, session.UserID).Error; err != nil {
			httpx.Unauthorized(c, "Invalid or expired session")
			c.Abort()
			return
		}
		if !user.Active {
			httpx.Forbidden(c, "Account is inactive")
			c.Abort()
			return
		}

		updates := map[string]any{"last_active_at": now}
		if user.Role == "admin" {
			updates["expires_at"] = now.Add(s.Config.SessionTTLAdmin)
		}
		_ = s.DB.WithContext(c.Request.Context()).Model(&models.Session{}).Where("id = ?", session.ID).Updates(updates)

		c.Set(contextUserKey, &user)
		c.Set(contextSessionKey, &session)

		if user.Role == "member" {
			var member models.Member
			if err := s.DB.WithContext(c.Request.Context()).Where("user_id = ? AND deleted_at IS NULL", user.ID).First(&member).Error; err == nil {
				c.Set(contextMemberKey, &member)
			}
		}
		if user.Role == "admin" {
			var admin models.Admin
			if err := s.DB.WithContext(c.Request.Context()).Where("user_id = ?", user.ID).First(&admin).Error; err == nil {
				c.Set(contextAdminKey, &admin)
			}
		}
		c.Next()
	}
}

func (s *Server) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := currentUser(c)
		if user == nil || user.Role != "admin" {
			httpx.Forbidden(c, "Administrator access is required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireDevice authenticates an ESP32/biometric device from X-Device-Token
// (or a raw X-Device-Api-Key) and stores it in the request context.
func (s *Server) RequireDevice() gin.HandlerFunc {
	return func(c *gin.Context) {
		device, ok := s.authenticateDeviceOrFail(c)
		if !ok {
			c.Abort()
			return
		}
		c.Set(contextDeviceKey, device)
		c.Next()
	}
}

// OptionalDevice authenticates a device when credentials are present but never
// rejects the request. It is used by routes that also support HMAC-only
// submissions without a device token.
func (s *Server) OptionalDevice() gin.HandlerFunc {
	return func(c *gin.Context) {
		if device, ok := s.authenticateDevice(c); ok {
			c.Set(contextDeviceKey, device)
		}
		c.Next()
	}
}

func (s *Server) RequireMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := currentUser(c)
		if user == nil || user.Role != "member" {
			httpx.Forbidden(c, "Member access is required")
			c.Abort()
			return
		}
		if currentMember(c) == nil {
			httpx.Forbidden(c, "Member profile is unavailable")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) RequirePrimaryAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := currentAdmin(c)
		if admin == nil || !admin.IsPrimary {
			httpx.Forbidden(c, "Primary administrator access is required")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) RequirePasswordChangeCompleted() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := currentUser(c)
		if user == nil || user.Role == "admin" {
			c.Next()
			return
		}
		member := currentMember(c)
		if member != nil && member.MustChangePassword {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success":              false,
				"message":              "You must change your password before accessing this feature",
				"must_change_password": true,
			})
			return
		}
		c.Next()
	}
}

func (s *Server) RateLimit(key string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		bucket := key + ":" + c.ClientIP()
		if !s.Limiter.Allow(bucket, limit, window) {
			c.Header("Retry-After", "60")
			httpx.Error(c, http.StatusTooManyRequests, "Too many requests. Please try again later.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func currentUser(c *gin.Context) *models.User {
	value, exists := c.Get(contextUserKey)
	if !exists {
		return nil
	}
	user, _ := value.(*models.User)
	return user
}

func currentMember(c *gin.Context) *models.Member {
	value, exists := c.Get(contextMemberKey)
	if !exists {
		return nil
	}
	member, _ := value.(*models.Member)
	return member
}

func currentDevice(c *gin.Context) *models.DeviceRegistration {
	value, exists := c.Get(contextDeviceKey)
	if !exists {
		return nil
	}
	device, _ := value.(*models.DeviceRegistration)
	return device
}

func currentAdmin(c *gin.Context) *models.Admin {
	value, exists := c.Get(contextAdminKey)
	if !exists {
		return nil
	}
	admin, _ := value.(*models.Admin)
	return admin
}

func currentSession(c *gin.Context) *models.Session {
	value, exists := c.Get(contextSessionKey)
	if !exists {
		return nil
	}
	session, _ := value.(*models.Session)
	return session
}

func (s *Server) audit(c *gin.Context, action, resourceType string, resourceID *int64, details any) {
	user := currentUser(c)
	var userID *int64
	if user != nil {
		value := user.ID
		userID = &value
	}
	encoded, _ := json.Marshal(details)
	if len(encoded) == 0 {
		encoded = []byte("{}")
	}
	ip := c.ClientIP()
	entry := models.AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      datatypes.JSON(encoded),
		IPAddress:    &ip,
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&entry).Error; err != nil {
		s.Log.Error("audit log write failed", "action", action, "error", err)
	}
}
