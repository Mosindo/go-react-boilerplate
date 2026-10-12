// Package httpx holds small helpers shared by Gin handlers.
package httpx

import (
	"net/http"
	"regexp"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// Fail writes a JSON error body.
func Fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// Internal logs the underlying error and answers with a generic 500.
func Internal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, "internal error")
}

// BindJSON binds the body or answers 400.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, http.StatusBadRequest, "invalid request")
		return false
	}
	return true
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether v is a canonical UUID string.
func IsUUID(v string) bool { return uuidRe.MatchString(v) }

// UUIDParam reads a path parameter that must be a UUID, answering 404 otherwise.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	v := c.Param(name)
	if !IsUUID(v) {
		Fail(c, http.StatusNotFound, "not found")
		return "", false
	}
	return strings.ToLower(v), true
}
