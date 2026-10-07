// Package httpx holds the small helpers shared by every feature handler so that
// handlers stay thin and error payloads stay consistent.
package httpx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

const maxJSONBodyBytes = 64 * 1024

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes the standard error payload.
func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorBody{Error: message, Code: code})
}

// Respond maps a service error to an HTTP response. Unexpected errors are logged
// and surfaced as a generic 500 so internals never leak.
func Respond(c *gin.Context, operation string, err error) {
	if appErr, ok := platformerrors.As(err); ok {
		logger.LogHandlerError(c, operation, appErr.Status, err)
		Fail(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, platformerrors.CodeInternal, "something went wrong")
}

// BindJSON decodes and validates a bounded JSON body. It writes the 400 response itself.
func BindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)
	if err := c.ShouldBindJSON(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			Fail(c, http.StatusRequestEntityTooLarge, platformerrors.CodeValidation, "request body too large")
			return false
		}
		logger.LogHandlerError(c, "bind", http.StatusBadRequest, err)
		Fail(c, http.StatusBadRequest, platformerrors.CodeValidation, "invalid request")
		return false
	}
	return true
}

// UserID returns the authenticated user id set by middleware.RequireUser.
func UserID(c *gin.Context) string { return c.GetString("userID") }

// UUIDParam reads a path parameter and validates it is a UUID; it writes the 404 itself.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		Fail(c, http.StatusNotFound, platformerrors.CodeNotFound, "not found")
		return "", false
	}
	return strings.ToLower(value), true
}

// Limit parses a `limit` query value into [1,max], falling back to def.
func Limit(raw string, def, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// Offset parses a non-negative `offset` query value (capped to keep scans bounded).
func Offset(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 {
		return 0
	}
	if n > 100000 {
		return 100000
	}
	return n
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
		default:
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}
	return true
}

// Items is the standard list envelope.
func Items[T any](items []T) gin.H {
	if items == nil {
		items = []T{}
	}
	return gin.H{"items": items}
}
