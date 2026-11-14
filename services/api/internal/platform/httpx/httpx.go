// Package httpx holds small HTTP helpers shared by every feature handler so
// that error payloads, pagination and identifier validation stay consistent.
package httpx

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorBody is the single error envelope returned by the API.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether value is a canonical UUID string.
func IsUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// Fail writes an error envelope with the given status and machine code.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: message, Code: code})
}

// BadRequest writes a 400 validation error.
func BadRequest(c *gin.Context, code, message string) {
	Fail(c, http.StatusBadRequest, code, message)
}

// Internal logs err and writes a generic 500 without leaking details.
func Internal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, "internal_error", "une erreur interne est survenue")
}

// UUIDParam reads a path parameter and validates it is a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.TrimSpace(c.Param(name))
	if !IsUUID(value) {
		BadRequest(c, "invalid_id", "identifiant invalide")
		return "", false
	}
	return strings.ToLower(value), true
}

// Limit parses ?limit= bounded to [1, max] with fallback when absent/invalid.
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

// Offset parses ?offset= as a non-negative integer bounded to max.
func Offset(c *gin.Context, max int) int {
	raw := strings.TrimSpace(c.Query("offset"))
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return 0
	}
	if parsed > max {
		return max
	}
	return parsed
}

// BindJSON decodes the body into dst and writes a 400 on failure.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		logger.LogHandlerError(c, "bind_json", http.StatusBadRequest, err)
		BadRequest(c, "invalid_request", "requête invalide")
		return false
	}
	return true
}
