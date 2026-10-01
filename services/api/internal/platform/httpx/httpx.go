// Package httpx holds small helpers shared by every feature handler.
package httpx

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// Error writes the API's uniform error envelope: {"error": message, "code": code}.
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}

// Internal logs err and answers with a generic 500 that never leaks details.
func Internal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Error(c, http.StatusInternalServerError, "internal", "une erreur interne est survenue")
}

// BadRequest answers 400 with the given validation message.
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "invalid_request", message)
}

// UserID returns the authenticated user id set by middleware.RequireUser.
func UserID(c *gin.Context) string { return c.GetString("userID") }

// Limit parses ?limit= clamped to [1, max], falling back to def.
func Limit(c *gin.Context, def, max int) int {
	n, err := strconv.Atoi(c.Query("limit"))
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// Offset parses ?offset= (>= 0, capped to avoid absurd scans).
func Offset(c *gin.Context) int {
	n, err := strconv.Atoi(c.Query("offset"))
	if err != nil || n < 0 {
		return 0
	}
	if n > 10_000 {
		return 10_000
	}
	return n
}

// Cursor parses a ?before= RFC3339 timestamp cursor. ok is false when absent; err when malformed.
func Cursor(c *gin.Context, key string) (t time.Time, ok bool, err error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return time.Time{}, false, nil
	}
	t, err = time.Parse(time.RFC3339Nano, raw)
	return t, err == nil, err
}

// IsUUID reports whether s is a canonical UUID. Handlers use it to turn malformed ids into 404/400
// before they reach the database.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if s[i] != '-' {
				return false
			}
		case !(s[i] >= '0' && s[i] <= '9' || s[i] >= 'a' && s[i] <= 'f' || s[i] >= 'A' && s[i] <= 'F'):
			return false
		}
	}
	return true
}
