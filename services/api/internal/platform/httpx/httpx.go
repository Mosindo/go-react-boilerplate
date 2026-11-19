// Package httpx contains small helpers shared by every HTTP handler:
// consistent error payloads, identity extraction, and input parsing.
package httpx

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorResponse is the single error shape returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes an error payload and aborts the request.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: code})
}

// Internal logs err and answers with a generic 500 that never leaks internals.
func Internal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, "internal_error", "something went wrong, please retry")
}

// BadRequest answers with a 400 validation error.
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, "invalid_request", message)
}

// UserID returns the authenticated user id set by middleware.RequireUser.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// SessionID returns the authenticated session id set by middleware.RequireUser.
func SessionID(c *gin.Context) string {
	return c.GetString("sessionID")
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether value is a canonical UUID string.
func IsUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

// UUIDParam reads a path parameter and validates it as a UUID. It writes a 400 and
// returns ok=false when the value is invalid.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		BadRequest(c, name+" must be a valid id")
		return "", false
	}
	return strings.ToLower(value), true
}

// Limit parses the "limit" query parameter, falling back to def and rejecting values
// outside [1, max].
func Limit(c *gin.Context, def, max int) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return def, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > max {
		BadRequest(c, "limit must be between 1 and "+strconv.Itoa(max))
		return 0, false
	}
	return value, true
}
