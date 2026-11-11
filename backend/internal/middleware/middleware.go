package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"gorm.io/datatypes"
)

// ResponseError represents a standardized API error response
type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// WriteError writes an error response to the HTTP client
func WriteError(c *gin.Context, status int, code, message string, details ...any) {
	errResponse := ResponseError{
		Code:    code,
		Message: message,
	}

	if len(details) > 0 {
		errResponse.Details = details[0]
	}

	c.Header("Content-Type", "application/json")
	c.Status(status)
	if err := json.NewEncoder(c.Writer).Encode(errResponse); err != nil {
		c.Error(err)
	}
}

// WriteSuccess writes a standardized success response
func WriteSuccess(c *gin.Context, status int, data any) {
	c.Header("Content-Type", "application/json")
	c.Status(status)
	if err := json.NewEncoder(c.Writer).Encode(map[string]any{
		"success": true,
		"data":    data,
	}); err != nil {
		c.Error(err)
	}
}

// AuthMiddleware provides authentication via Bearer token
func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			WriteError(c, http.StatusUnauthorized, "auth_required",
				"Authentication token is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			WriteError(c, http.StatusUnauthorized, "auth_invalid_format",
				"Invalid authentication header format")
			c.Abort()
			return
		}

		token := parts[1]
		if token == "" {
			WriteError(c, http.StatusUnauthorized, "auth_empty_token",
				"Authentication token is empty")
			c.Abort()
			return
		}

		// Set user in context - in full implementation, validate token
		c.Set("current_user", &models.User{
			ID:        1,
			Email:     "placeholder",
			Role:      "member",
			Active:    true,
		})

		c.Next()
	}
}

// RequireAuth is a Gin handler middleware that requires authentication
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("current_user")
		if !exists || user == nil {
			WriteError(c, http.StatusUnauthorized, "auth_required",
				"Authentication required")
			c.Abort()
			return
		}

		c.Set("current_user", user)
		c.Next()
	}
}

// RequireAdmin is a Gin handler middleware that requires administrator role
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("current_user")
		if !exists || user == nil {
			WriteError(c, http.StatusUnauthorized, "auth_required",
				"Authentication required")
			c.Abort()
			return
		}

		// Check if user has admin role
		u, ok := user.(*models.User)
		if !ok || u.Role != "admin" {
			WriteError(c, http.StatusForbidden, "admin_required",
				"Administrator access is required")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePrimaryAdmin is a Gin handler middleware that requires primary administrator
func RequirePrimaryAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, exists := c.Get("current_admin")
		if !exists || admin == nil {
			WriteError(c, http.StatusForbidden, "admin_required",
				"Administrator access is required")
			c.Abort()
			return
		}

		// Check if this is a primary admin
		a, ok := admin.(*models.Admin)
		if !ok || !a.IsPrimary {
			WriteError(c, http.StatusForbidden, "primary_admin_required",
				"Primary administrator access is required")
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuditLogMiddleware logs all requests for audit purposes
func AuditLogMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)

		// Log the request for audit purposes
		user, _ := c.Get("current_user")
		var userID *int64
		if u, ok := user.(*models.User); ok && u.ID != 0 {
			userID = &u.ID
		}

		ip := c.ClientIP()
		action := fmt.Sprintf("http_%s_%d", c.Request.Method, c.Writer.Status())

		details := map[string]any{
			"path":        c.Request.URL.Path,
			"method":      c.Request.Method,
			"status":      c.Writer.Status(),
			"duration_ms": duration.Milliseconds(),
			"ip_address":  ip,
		}
		encodedDetails, _ := json.Marshal(details)

		auditEntry := models.AuditLog{
			ID:           0,
			UserID:       userID,
			Action:       action,
			ResourceType: "api_endpoint",
			ResourceID:   nil,
			Details:      datatypes.JSON(encodedDetails),
			IPAddress:    &ip,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Don't fail if audit log write fails
		db.Create(&auditEntry)
	}
}