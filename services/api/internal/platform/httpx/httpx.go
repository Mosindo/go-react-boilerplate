// Package httpx holds the tiny helpers shared by all feature handlers so that
// every endpoint answers errors and pagination in the same shape.
package httpx

import (
	"net/http"
	"strconv"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// Fail writes {"error": "..."} using the status carried by the error (500 otherwise) and logs it.
func Fail(c *gin.Context, operation string, err error) {
	status, message := platformerrors.Status(err)
	if status >= 500 {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.JSON(status, gin.H{"error": message})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": message})
}

// Limit parses ?limit= with a default and a hard maximum; invalid values fall back to the default.
func Limit(c *gin.Context, def, max int) int {
	raw := c.Query("limit")
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
