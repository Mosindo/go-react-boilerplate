// Package httpx contains small, dependency-free helpers shared by feature
// handlers: consistent error payloads, identifier validation and pagination.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorResponse is the error envelope returned by every endpoint.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// Fail writes a JSON error and logs it with the request context.
func Fail(c *gin.Context, operation string, status int, code, message string, err error) {
	if err != nil {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: code})
}

// BadRequest is a shorthand for validation failures.
func BadRequest(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: message, Code: code})
}

// Internal hides internal details from clients while logging them.
func Internal(c *gin.Context, operation string, err error) {
	Fail(c, operation, http.StatusInternalServerError, "internal_error", "something went wrong, please retry", err)
}

// UserID returns the authenticated user id set by middleware.RequireUser.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// IsUUID reports whether v is a canonical 36-character UUID string.
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

// Limit parses a positive page size, clamped to max.
func Limit(raw string, fallback, max int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

// Offset parses a non-negative offset, clamped to max to avoid deep scans.
func Offset(raw string, max int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 0 {
		return 0
	}
	if parsed > max {
		return max
	}
	return parsed
}
