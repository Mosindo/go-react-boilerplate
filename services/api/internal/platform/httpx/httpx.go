// Package httpx holds small HTTP helpers shared by every feature handler:
// a consistent error envelope, request-scoped identity accessors and strict
// parsing of path/query parameters.
package httpx

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// ErrorBody is the single error envelope returned by the API.
type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// Fail writes the error envelope, aborts the chain and logs the cause.
func Fail(c *gin.Context, status int, code, message string, cause error) {
	if cause == nil {
		cause = errors.New(code)
	}
	logger.LogHandlerError(c, c.FullPath(), status, cause)
	c.AbortWithStatusJSON(status, ErrorBody{Error: message, Code: code})
}

// BadRequest is a shorthand for validation failures.
func BadRequest(c *gin.Context, code, message string) {
	Fail(c, http.StatusBadRequest, code, message, nil)
}

// Internal hides internal failures behind a generic message.
func Internal(c *gin.Context, err error) {
	Fail(c, http.StatusInternalServerError, "internal_error", "Une erreur est survenue, réessaie plus tard.", err)
}

// NotFound is used for missing resources and for resources the caller may not
// see, so that existence is never leaked.
func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, "not_found", message, nil)
}

// UserID returns the authenticated user id set by the auth middleware.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// OrganizationID returns the tenant id set by the auth middleware.
func OrganizationID(c *gin.Context) string {
	return c.GetString("organizationID")
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
		BadRequest(c, "invalid_id", "Identifiant invalide.")
		return "", false
	}
	return value, true
}

// Limit parses ?limit= strictly: absent means fallback, invalid or out of
// range values are rejected instead of silently clamped.
func Limit(c *gin.Context, fallback, max int) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return fallback, true
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 || parsed > max {
		BadRequest(c, "invalid_limit", "Paramètre limit invalide.")
		return 0, false
	}
	return parsed, true
}

// Cursor is an opaque keyset pagination position (timestamp + tie-breaker id).
type Cursor struct {
	At time.Time
	ID string
}

// EncodeCursor serialises a cursor into an opaque URL-safe token.
func EncodeCursor(at time.Time, id string) string {
	raw := strconv.FormatInt(at.UTC().UnixNano(), 10) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor reads ?cursor=. A missing cursor yields (nil, true).
func DecodeCursor(c *gin.Context) (*Cursor, bool) {
	raw := strings.TrimSpace(c.Query("cursor"))
	if raw == "" {
		return nil, true
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		BadRequest(c, "invalid_cursor", "Curseur de pagination invalide.")
		return nil, false
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 || !IsUUID(parts[1]) {
		BadRequest(c, "invalid_cursor", "Curseur de pagination invalide.")
		return nil, false
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || nanos <= 0 {
		BadRequest(c, "invalid_cursor", "Curseur de pagination invalide.")
		return nil, false
	}
	return &Cursor{At: time.Unix(0, nanos).UTC(), ID: parts[1]}, true
}

// BindJSON binds a JSON body and writes a uniform 400 on failure.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, http.StatusBadRequest, "invalid_request", "Requête invalide.", err)
		return false
	}
	return true
}

// NoStore marks a response as never cacheable (tokens, private data).
func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}

// ValidationError is returned by services for invalid user input; handlers
// translate it to a 400 with a stable machine code and a readable message.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Code + ": " + e.Message }

// Invalid builds a ValidationError.
func Invalid(code, message string) error {
	return &ValidationError{Code: code, Message: message}
}

// WriteValidation writes a 400 when err is a ValidationError.
func WriteValidation(c *gin.Context, err error) bool {
	var v *ValidationError
	if errors.As(err, &v) {
		Fail(c, http.StatusBadRequest, v.Code, v.Message, err)
		return true
	}
	return false
}
