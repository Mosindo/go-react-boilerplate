// Package httpx contains small helpers shared by every feature handler so that
// error payloads, identifiers and pagination are handled consistently.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorBody is the JSON shape of every error response: {"error": "...", "code": "..."}.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes a client error with a stable machine-readable code.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: message, Code: code})
}

// BadRequest is a shortcut for validation errors.
func BadRequest(c *gin.Context, code, message string) {
	Fail(c, http.StatusBadRequest, code, message)
}

// Internal logs the underlying error and returns a generic 500 payload.
func Internal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, "internal_error", "something went wrong, please try again")
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// IsUUID validates the canonical 8-4-4-4-12 hexadecimal representation.
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

// UUIDParam reads a path parameter and rejects anything that is not a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(value) {
		BadRequest(c, "invalid_id", "invalid "+name)
		return "", false
	}
	return value, true
}

// Limit parses ?limit= bounded to [1, max] with a fallback for empty/invalid input.
func Limit(c *gin.Context, fallback, max int) int {
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

// NoStore marks a response as non-cacheable (private user data).
func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}
