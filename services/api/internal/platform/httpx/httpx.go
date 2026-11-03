// Package httpx holds small Gin helpers shared by feature handlers.
package httpx

import (
	"net/http"
	"strconv"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Fail writes a consistent JSON error. Business errors keep their status and
// code; anything else is logged and reported as a generic 500.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		logger.LogHandlerError(c, operation, appErr.Status, err)
		c.AbortWithStatusJSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal"})
}

// BindJSON decodes the body and reports invalid payloads as 400.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, "bind", platformerrors.Invalid("invalid request"))
		return false
	}
	return true
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string { return c.GetString("userID") }

// UUIDParam validates a path parameter as UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := c.Param(name)
	if _, err := uuid.Parse(value); err != nil {
		Fail(c, "param."+name, platformerrors.Invalid("invalid "+name))
		return "", false
	}
	return value, true
}

// Limit parses ?limit= with a default and a hard maximum.
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

// UUIDParamValue validates an already-extracted string as UUID.
func UUIDParamValue(value string) (string, bool) {
	if _, err := uuid.Parse(value); err != nil {
		return "", false
	}
	return value, true
}
