// Package httpx holds the shared HTTP conventions of the API: the error body
// {"error": "...", "code": "..."}, typed application errors, request binding
// helpers and pagination/cursor helpers.
package httpx

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Machine readable error codes (see docs/API.md).
const (
	CodeInvalidRequest    = "invalid_request"
	CodeUnauthorized      = "unauthorized"
	CodeForbidden         = "forbidden"
	CodeNotFound          = "not_found"
	CodeConflict          = "conflict"
	CodeRateLimited       = "rate_limited"
	CodeProfileIncomplete = "profile_incomplete"
	CodeUnderage          = "underage"
	CodeBlocked           = "blocked"
	CodePayloadTooLarge   = "payload_too_large"
	CodeUnsupportedMedia  = "unsupported_media"
	CodeInternal          = "internal"
)

// Error is an application error carrying its HTTP status and machine code.
// Services return these for expected failures; anything else becomes a 500.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func newErr(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

func BadRequest(msg string) *Error { return newErr(http.StatusBadRequest, CodeInvalidRequest, msg) }
func Unprocessable(msg string) *Error {
	return newErr(http.StatusUnprocessableEntity, CodeInvalidRequest, msg)
}
func Unauthorized(msg string) *Error { return newErr(http.StatusUnauthorized, CodeUnauthorized, msg) }
func Forbidden(msg string) *Error    { return newErr(http.StatusForbidden, CodeForbidden, msg) }
func NotFound(msg string) *Error     { return newErr(http.StatusNotFound, CodeNotFound, msg) }
func Conflict(msg string) *Error     { return newErr(http.StatusConflict, CodeConflict, msg) }
func ProfileIncomplete() *Error {
	return newErr(http.StatusConflict, CodeProfileIncomplete, "complete your profile (details, location and at least one photo) first")
}
func Underage() *Error {
	return newErr(http.StatusUnprocessableEntity, CodeUnderage, "you must be at least 18 years old")
}
func Blocked() *Error {
	return newErr(http.StatusForbidden, CodeBlocked, "this conversation is not available")
}
func PayloadTooLarge(msg string) *Error {
	return newErr(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, msg)
}
func UnsupportedMedia(msg string) *Error {
	return newErr(http.StatusUnsupportedMediaType, CodeUnsupportedMedia, msg)
}

// Abort writes the standard error body and stops the handler chain.
func Abort(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg, "code": code})
}

// Fail maps err to a response. Unknown errors are logged and returned as an
// opaque 500 so internals never leak to clients.
func Fail(c *gin.Context, err error) {
	var appErr *Error
	if errors.As(err, &appErr) {
		Abort(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		Abort(c, http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "request body too large")
		return
	}
	log.Printf(`{"event":"internal_error","request_id":%q,"method":%q,"path":%q,"error":%q}`,
		c.GetString("request_id"), c.Request.Method, c.FullPath(), sanitize(err.Error()))
	Abort(c, http.StatusInternalServerError, CodeInternal, "internal error")
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}

// BindJSON decodes exactly one JSON object from the (size limited) body.
func BindJSON(c *gin.Context, dst any) error {
	dec := json.NewDecoder(c.Request.Body)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return PayloadTooLarge("request body too large")
		}
		return BadRequest("invalid JSON body")
	}
	if dec.More() {
		return BadRequest("invalid JSON body")
	}
	return nil
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical textual UUID.
func IsUUID(s string) bool { return uuidRE.MatchString(s) }

// NormalizeUUID lower-cases a valid UUID; ok is false for anything else.
func NormalizeUUID(s string) (string, bool) {
	if !IsUUID(s) {
		return "", false
	}
	return strings.ToLower(s), true
}

// UserID returns the authenticated user id set by the RequireUser middleware.
func UserID(c *gin.Context) string { return c.GetString("userID") }

// SessionID returns the session id of the access token.
func SessionID(c *gin.Context) string { return c.GetString("sessionID") }

// ParseLimit validates the `limit` query parameter.
func ParseLimit(c *gin.Context, def, max int) (int, error) {
	raw := c.Query("limit")
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		return 0, BadRequest("limit must be an integer between 1 and " + strconv.Itoa(max))
	}
	return n, nil
}

// EncodeCursor / DecodeCursor turn a JSON-serialisable keyset position into an
// opaque URL-safe string and back.
func EncodeCursor(v any) string {
	b, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(b)
}

func DecodeCursor(raw string, dst any) error {
	if len(raw) > 512 {
		return BadRequest("invalid cursor")
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return BadRequest("invalid cursor")
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return BadRequest("invalid cursor")
	}
	return nil
}

// Page is the standard list envelope.
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
