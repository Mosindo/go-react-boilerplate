// Package httpx holds small HTTP helpers shared by feature handlers.
package httpx

import (
	"net/http"
	"strconv"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// Fail logs err and writes a JSON error body.
func Fail(c *gin.Context, op string, status int, msg string, err error) {
	if err != nil {
		logger.LogHandlerError(c, op, status, err)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

// Internal writes a generic 500 and logs the cause.
func Internal(c *gin.Context, op string, err error) {
	Fail(c, op, http.StatusInternalServerError, "internal error", err)
}

// BadRequest writes a 400 with a client-safe message.
func BadRequest(c *gin.Context, op, msg string, err error) {
	Fail(c, op, http.StatusBadRequest, msg, err)
}

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

// NoStore marks a response as non-cacheable (private data).
func NoStore(c *gin.Context) { c.Header("Cache-Control", "no-store") }
