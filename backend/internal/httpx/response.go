package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Unauthorized returns HTTP 401 with error response
func Unauthorized(c *gin.Context, message string) {
	WriteError(c, http.StatusUnauthorized, "unauthorized", message)
}

// Forbidden returns HTTP 403 with error response
func Forbidden(c *gin.Context, message string) {
	WriteError(c, http.StatusForbidden, "forbidden", message)
}

// NotFound returns HTTP 404 with error response
func NotFound(c *gin.Context, message string) {
	WriteError(c, http.StatusNotFound, "not_found", message)
}

// Error returns HTTP 500 with error response
func Error(c *gin.Context, status int, message string) {
	WriteError(c, status, "internal_error", message)
}

// OK returns HTTP 200 with success response
func OK(c *gin.Context, data any) {
	WriteSuccess(c, http.StatusOK, data)
}

// Created returns HTTP 201 with success response
func Created(c *gin.Context, data any) {
	WriteSuccess(c, http.StatusCreated, data)
}

// WriteError writes a standardized error JSON response
func WriteError(c *gin.Context, status int, code, message string) {
	c.Header("Content-Type", "application/json")
	c.Status(status)
	if err := json.NewEncoder(c.Writer).Encode(map[string]any{
		"success": false,
		"error":   code,
		"message": message,
	}); err != nil {
		// If we can't encode the error, just log it
		_ = err
	}
}

// WriteSuccess writes a standardized success JSON response
func WriteSuccess(c *gin.Context, status int, data any) {
	c.Header("Content-Type", "application/json")
	c.Status(status)
	if err := json.NewEncoder(c.Writer).Encode(map[string]any{
		"success": true,
		"data":    data,
	}); err != nil {
		_ = err
	}
}

// ExtractIDFromString extracts an int64 ID from a string ID parameter
func ExtractIDFromString(id string) (int64, error) {
	var numericID int64
	_, err := fmt.Sscanf(id, "%d", &numericID)
	return numericID, err
}