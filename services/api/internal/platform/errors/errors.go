package platformerrors

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// Error is a client-facing error carrying an HTTP status and a stable machine code.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(code, message string) *Error { return New(http.StatusBadRequest, code, message) }
func Unauthorized(code, message string) *Error {
	return New(http.StatusUnauthorized, code, message)
}
func Forbidden(code, message string) *Error { return New(http.StatusForbidden, code, message) }
func NotFound(code, message string) *Error  { return New(http.StatusNotFound, code, message) }
func Conflict(code, message string) *Error  { return New(http.StatusConflict, code, message) }

// Respond writes a consistent JSON error body: {"error": message, "code": code}.
// Unknown errors are logged and masked as a 500.
func Respond(c *gin.Context, err error) {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		c.AbortWithStatusJSON(apiErr.Status, gin.H{"error": apiErr.Message, "code": apiErr.Code})
		return
	}
	log.Printf(`{"event":"internal_error","request_id":%q,"path":%q,"error":%q}`,
		c.GetString("request_id"), c.FullPath(), err.Error())
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal"})
}

// RespondBind answers with a 400 for malformed or invalid request bodies.
func RespondBind(c *gin.Context, err error) {
	Respond(c, BadRequest("invalid_request", "invalid request"))
	_ = err
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical UUID string.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }

// ParamUUID reads a path parameter and answers 404 when it is not a valid UUID,
// so malformed ids never reach the database.
func ParamUUID(c *gin.Context, name string) (string, bool) {
	v := strings.ToLower(c.Param(name))
	if !IsUUID(v) {
		Respond(c, NotFound("not_found", "not found"))
		return "", false
	}
	return v, true
}
