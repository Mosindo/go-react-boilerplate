// Package httpx holds small HTTP helpers shared by every feature handler:
// a consistent error envelope, authenticated user lookup and input parsing.
package httpx

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the single error envelope returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes the error envelope and aborts the handler chain.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: code})
}

// UserID returns the authenticated user id set by middleware.RequireUser.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// Limit parses a positive page size, clamped to [1, max].
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

// Offset parses a non-negative offset, clamped to max.
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

// IsUUID reports whether v is a canonical 8-4-4-4-12 hex UUID.
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

// UUIDParam reads a path parameter and validates it is a UUID. On failure it
// writes a 400 response and returns ok=false.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(v) {
		Fail(c, 400, "invalid_id", "invalid "+name)
		return "", false
	}
	return v, true
}
