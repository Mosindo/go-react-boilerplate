// Package httpx holds small HTTP helpers shared by feature handlers so that
// every endpoint parses pagination, identifiers and errors the same way.
package httpx

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrorBody is the JSON error envelope returned by every endpoint.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// Error writes a JSON error response with a stable machine-readable code.
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorBody{Error: message, Code: code})
}

// AbortError is like Error but also aborts the middleware chain.
func AbortError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: message, Code: code})
}

// InvalidRequest is the canonical 400 response for malformed payloads.
func InvalidRequest(c *gin.Context) {
	Error(c, http.StatusBadRequest, "invalid_request", "invalid request")
}

// Internal is the canonical 500 response; details are only logged.
func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, "internal_error", message)
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// OrganizationID returns the authenticated tenant id set by the auth middleware.
func OrganizationID(c *gin.Context) string {
	return c.GetString("organizationID")
}

// ParseLimit parses a positive page size, falling back to def and capping at max.
func ParseLimit(raw string, def, max int) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return def
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return def
	}
	if parsed > max {
		return max
	}
	return parsed
}

// ParseOffset parses a non-negative offset, capped to keep queries cheap.
func ParseOffset(raw string) int {
	const maxOffset = 10_000
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed < 0 {
		return 0
	}
	if parsed > maxOffset {
		return maxOffset
	}
	return parsed
}

// IsUUID reports whether v is a canonical 36-character UUID string.
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

// UUIDParam returns the lower-cased path parameter when it is a valid UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(value) {
		return "", false
	}
	return value, true
}
