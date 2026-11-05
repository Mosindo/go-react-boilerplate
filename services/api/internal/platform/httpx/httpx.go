// Package httpx holds small Gin helpers shared by all features.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Error writes the standard error envelope: {"error": "<message>"}.
func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// BadRequest writes a 400 with the given message.
func BadRequest(c *gin.Context, message string) { Error(c, http.StatusBadRequest, message) }

// Internal writes a generic 500 (details never leak to clients).
func Internal(c *gin.Context) { Error(c, http.StatusInternalServerError, "internal error") }

// Bind decodes a JSON body, answering 400 on failure.
func Bind(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	if err := c.ShouldBindJSON(dst); err != nil {
		BadRequest(c, "invalid request")
		return false
	}
	return true
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string { return c.GetString("userID") }

// Limit parses ?limit with a default and an upper bound.
func Limit(c *gin.Context, def, max int) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		BadRequest(c, "invalid limit")
		return 0, false
	}
	return n, true
}
