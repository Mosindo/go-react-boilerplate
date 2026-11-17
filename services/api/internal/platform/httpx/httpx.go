// Package httpx holds small HTTP helpers shared by every feature handler so error
// payloads, identifiers and pagination are validated the same way everywhere.
package httpx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorBody is the JSON shape of every error response.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes a consistent error payload and aborts the request.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: message, Code: code})
}

// FailInternal logs the underlying error and returns a generic 500.
func FailInternal(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
	Fail(c, http.StatusInternalServerError, "internal_error", "Une erreur inattendue est survenue.")
}

// BindJSON decodes the body into dst, answering 400 on malformed input.
func BindJSON(c *gin.Context, operation string, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		logger.LogHandlerError(c, operation+".bind", http.StatusBadRequest, err)
		Fail(c, http.StatusBadRequest, "invalid_request", "Requête invalide.")
		return false
	}
	return true
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// UUIDParam validates a path parameter as a canonical UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(value) {
		Fail(c, http.StatusBadRequest, "invalid_id", "Identifiant invalide.")
		return "", false
	}
	return value, true
}

// IsUUID reports whether v is a hyphenated, 36-character UUID.
func IsUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i := 0; i < len(v); i++ {
		ch := v[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
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

var ErrInvalidLimit = errors.New("invalid limit")

// Limit parses ?limit= strictly: absent means fallback, invalid or out-of-range values are rejected.
func Limit(c *gin.Context, fallback, max int) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return fallback, true
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 || parsed > max {
		Fail(c, http.StatusBadRequest, "invalid_limit", "Paramètre limit invalide.")
		return 0, false
	}
	return parsed, true
}

// OptionalUUIDQuery validates an optional UUID query parameter (used for cursors).
func OptionalUUIDQuery(c *gin.Context, name string) (string, bool) {
	raw := strings.ToLower(strings.TrimSpace(c.Query(name)))
	if raw == "" {
		return "", true
	}
	if !IsUUID(raw) {
		Fail(c, http.StatusBadRequest, "invalid_cursor", "Curseur invalide.")
		return "", false
	}
	return raw, true
}
