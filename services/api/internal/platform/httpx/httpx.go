// Package httpx holds the small HTTP helpers shared by every feature handler:
// consistent error payloads, authenticated user lookup and query parsing.
package httpx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes an error response. Expected *AppError values are returned as-is;
// any other error becomes an opaque 500 and is logged with the operation name.
func Fail(c *gin.Context, operation string, err error) {
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		if appErr.Status >= http.StatusInternalServerError {
			logger.LogHandlerError(c, operation, appErr.Status, err)
		}
		c.AbortWithStatusJSON(appErr.Status, ErrorResponse{Error: appErr.Message, Code: appErr.Code})
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "internal error", Code: "internal_error"})
}

// BadRequest reports a malformed request body or parameter.
func BadRequest(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusBadRequest, err)
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request", Code: "invalid_request"})
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// UUIDParam reads a path parameter and validates it as a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "invalid " + name, Code: "invalid_request"})
		return "", false
	}
	return strings.ToLower(value), true
}

// QueryLimit parses a bounded positive page size.
func QueryLimit(c *gin.Context, fallback, max int) int {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

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
			continue
		}
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}
