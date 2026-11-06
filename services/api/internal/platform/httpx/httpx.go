// Package httpx holds small helpers shared by every feature handler so that
// responses, errors and pagination parameters stay consistent.
package httpx

import (
	"net/http"
	"strconv"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// Error writes the standard error envelope: {"error": "...", "code": "..."}.
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}

// Internal logs err and answers with a generic 500.
func Internal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Error(c, http.StatusInternalServerError, "internal_error", "something went wrong")
}

// BadRequest answers with a generic validation failure.
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "invalid_request", message)
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string { return c.GetString("userID") }

// Limit parses ?limit= within [1, max], using def when absent or invalid.
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

// NoStore marks a response as non-cacheable (tokens, private data).
func NoStore(c *gin.Context) { c.Header("Cache-Control", "no-store") }
