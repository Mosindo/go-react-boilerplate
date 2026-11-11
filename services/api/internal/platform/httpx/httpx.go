// Package httpx holds small helpers shared by every feature handler:
// consistent error payloads, request-scoped identity and input parsing.
package httpx

import (
	"net/http"
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

// Fail writes a JSON error and logs it with request context.
func Fail(c *gin.Context, status int, code, message string, err error) {
	if err != nil {
		logger.LogHandlerError(c, c.FullPath()+" "+code, status, err)
	}
	c.AbortWithStatusJSON(status, ErrorResponse{Error: message, Code: code})
}

func BadRequest(c *gin.Context, code, message string) {
	Fail(c, http.StatusBadRequest, code, message, nil)
}

func Internal(c *gin.Context, err error) {
	Fail(c, http.StatusInternalServerError, "internal_error", "Une erreur inattendue est survenue.", err)
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString(ContextUserID)
}

func SessionID(c *gin.Context) string {
	return c.GetString(ContextSessionID)
}

const (
	ContextUserID    = "userID"
	ContextSessionID = "sessionID"
)

// Limit parses a bounded positive integer query parameter.
func Limit(c *gin.Context, fallback, max int) int {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	if v > max {
		return max
	}
	return v
}

// Offset parses a non-negative bounded offset query parameter.
func Offset(c *gin.Context, max int) int {
	raw := strings.TrimSpace(c.Query("offset"))
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return 0
	}
	if v > max {
		return max
	}
	return v
}

// IsUUID reports whether v is a canonical 8-4-4-4-12 hex UUID.
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

// UUIDParam reads a path parameter and rejects it with 400 when it is not a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(v) {
		BadRequest(c, "invalid_id", "Identifiant invalide.")
		return "", false
	}
	return v, true
}

// BindJSON binds the body and writes a uniform 400 on failure.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, http.StatusBadRequest, "invalid_request", "Requête invalide.", err)
		return false
	}
	return true
}
