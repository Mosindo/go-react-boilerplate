// Package httpx holds small, dependency-free helpers shared by feature handlers:
// identifier validation, pagination parsing and consistent JSON error bodies.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorBody is the single error envelope returned by every endpoint.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// Fail logs the underlying error and writes a sanitized JSON error.
func Fail(c *gin.Context, operation string, status int, message string, err error) {
	if err != nil {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.AbortWithStatusJSON(status, ErrorBody{Error: message})
}

// FailCode is Fail with a stable machine-readable code the client can branch on.
func FailCode(c *gin.Context, operation string, status int, code, message string, err error) {
	if err != nil {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.AbortWithStatusJSON(status, ErrorBody{Error: message, Code: code})
}

// BadRequest is a shortcut for validation failures.
func BadRequest(c *gin.Context, operation, message string, err error) {
	Fail(c, operation, http.StatusBadRequest, message, err)
}

// UserID returns the authenticated user id set by middleware.RequireUser.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// OrganizationID returns the tenant id set by middleware.RequireUser.
func OrganizationID(c *gin.Context) string {
	return c.GetString("organizationID")
}

// Limit parses a positive integer query value clamped to [1, max].
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

// Offset parses a non-negative integer query value clamped to max.
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

// IsUUID reports whether v is a canonical 8-4-4-4-12 hexadecimal UUID.
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

// UUIDParam extracts and validates a UUID path parameter. It writes a 400 and
// returns false when the value is malformed.
func UUIDParam(c *gin.Context, name, operation string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(value) {
		BadRequest(c, operation, "invalid "+name, nil)
		return "", false
	}
	return value, true
}
