// Package platformerrors defines the application error type shared by all
// features. Services return *Error values; handlers translate them to HTTP.
package platformerrors

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error is a client-safe application error.
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
func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, "unauthorized", message)
}
func Forbidden(message string) *Error      { return New(http.StatusForbidden, "forbidden", message) }
func NotFound(message string) *Error       { return New(http.StatusNotFound, "not_found", message) }
func Conflict(code, message string) *Error { return New(http.StatusConflict, code, message) }

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// Respond writes err as the JSON error envelope {"error": message, "code": code}.
// Unknown errors are logged by the caller and reported as a generic 500.
func Respond(c *gin.Context, err error) {
	var appErr *Error
	if errors.As(err, &appErr) {
		c.JSON(appErr.Status, gin.H{"error": appErr.Message, "code": appErr.Code})
		return
	}
	log.Printf(`{"event":"internal_error","request_id":%q,"path":%q,"error":%q}`, c.GetString("request_id"), c.Request.URL.Path, err.Error())
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal"})
}
