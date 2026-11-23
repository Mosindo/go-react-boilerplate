// Package httpx holds small, shared HTTP helpers used by every feature handler:
// consistent error payloads, identity extraction and query parsing.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorResponse is the single error shape returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes a consistent JSON error and logs the underlying cause.
func Fail(c *gin.Context, status int, code, message string, operation string, cause error) {
	if cause != nil {
		logger.LogHandlerError(c, operation, status, cause)
	}
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: code})
}

// BadRequest is a shortcut for validation errors.
func BadRequest(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: message, Code: code})
}

// Internal hides internal details from the client while logging them.
func Internal(c *gin.Context, operation string, cause error) {
	Fail(c, http.StatusInternalServerError, "internal_error", "something went wrong, please try again", operation, cause)
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// ParseLimit parses a positive bounded integer query value.
func ParseLimit(raw string, fallback, max int) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

// IsUUID reports whether v is a canonical textual UUID.
func IsUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i := 0; i < len(v); i++ {
		ch := v[i]
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
			continue
		}
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

// UUIDParam reads a path parameter and validates it as a UUID. It writes a 400
// and returns false when invalid.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(value) {
		BadRequest(c, "invalid_id", "invalid "+name)
		return "", false
	}
	return value, true
}
