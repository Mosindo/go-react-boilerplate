// Package httpx contains small Gin helpers shared by feature handlers:
// consistent error payloads, request binding and query parsing.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorResponse is the JSON shape of every error returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Abort writes an error payload and stops the handler chain.
func Abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: code})
}

// Fail maps err to an HTTP response. Application errors keep their status and
// message; anything else becomes a generic 500 so internals never leak.
func Fail(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		if appErr.Status >= http.StatusInternalServerError {
			logger.LogHandlerError(c, operation, appErr.Status, err)
		}
		Abort(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Abort(c, http.StatusInternalServerError, "internal_error", "something went wrong, please retry")
}

// BindJSON decodes the request body into dst and writes a 400 on failure.
func BindJSON(c *gin.Context, operation string, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		logger.LogHandlerError(c, operation+".bind", http.StatusBadRequest, err)
		Abort(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return false
	}
	return true
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// SessionID returns the authenticated session id set by the auth middleware.
func SessionID(c *gin.Context) string {
	return c.GetString("sessionID")
}

// UUIDParam reads a path parameter and validates it is a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		Abort(c, http.StatusBadRequest, "invalid_id", "invalid "+name)
		return "", false
	}
	return strings.ToLower(value), true
}

// Limit parses a page-size query parameter, clamped to [1, max].
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

// IsUUID reports whether v is a canonical 36-character UUID.
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
