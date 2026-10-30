// Package httpx holds small helpers shared by feature handlers.
package httpx

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// Fail logs err for operators and answers with a stable, client-safe message.
func Fail(c *gin.Context, operation string, status int, message string, err error) {
	if err != nil {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

func BadRequest(c *gin.Context, operation string, err error) {
	Fail(c, operation, http.StatusBadRequest, "invalid request", err)
}

func Internal(c *gin.Context, operation string, err error) {
	Fail(c, operation, http.StatusInternalServerError, "internal error", err)
}

// ParamUUID reads a path parameter and rejects anything that is not a UUID.
func ParamUUID(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return "", false
	}
	return strings.ToLower(value), true
}

// QueryLimit parses ?limit= with a default and an upper bound.
func QueryLimit(c *gin.Context, def, max int) int {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// QueryCursor parses ?before= as an RFC3339 timestamp. The zero time means "no cursor".
func QueryCursor(c *gin.Context) (time.Time, bool) {
	raw := strings.TrimSpace(c.Query("before"))
	if raw == "" {
		return time.Time{}, true
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid before cursor"})
		return time.Time{}, false
	}
	return t, true
}

// IsUUID reports whether value has the canonical 8-4-4-4-12 hexadecimal form.
func IsUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i := 0; i < len(value); i++ {
		switch i {
		case 8, 13, 18, 23:
			if value[i] != '-' {
				return false
			}
		default:
			ch := value[i]
			isHex := (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}

// WriteError maps validation failures to 400 and anything unexpected to 500.
// It returns true when it handled the error.
func WriteError(c *gin.Context, operation string, err error) bool {
	if err == nil {
		return false
	}
	if v, ok := platformerrors.AsValidation(err); ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": v.Message, "field": v.Field})
		return true
	}
	Internal(c, operation, err)
	return true
}
