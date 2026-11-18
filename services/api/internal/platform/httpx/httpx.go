// Package httpx holds small, dependency-free helpers shared by every feature
// handler: consistent error payloads, query parsing and identifier validation.
package httpx

import (
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

// Fail writes a JSON error and logs the underlying cause (sanitised).
func Fail(c *gin.Context, status int, code, message, operation string, cause error) {
	if cause != nil {
		logger.LogHandlerError(c, operation, status, cause)
	}
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: code})
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// OrganizationID returns the authenticated tenant id set by the auth middleware.
func OrganizationID(c *gin.Context) string {
	return c.GetString("organizationID")
}

// Limit parses a positive page size, falling back to def and clamping to max.
func Limit(raw string, def, max int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return def
	}
	if parsed > max {
		return max
	}
	return parsed
}

// Offset parses a non-negative offset, clamped to max to keep scans bounded.
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
		default:
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}
	return true
}

// NormalizeUUID lowercases a UUID so string comparisons match Postgres ordering.
func NormalizeUUID(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
