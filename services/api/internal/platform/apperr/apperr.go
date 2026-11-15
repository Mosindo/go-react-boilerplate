// Package apperr defines typed application errors that carry an HTTP status
// and a stable machine-readable code. Services return them; handlers map them
// to responses without re-implementing switch statements.
package apperr

import (
	"errors"
	"net/http"
)

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
func NotFound(code, message string) *Error   { return New(http.StatusNotFound, code, message) }
func Conflict(code, message string) *Error   { return New(http.StatusConflict, code, message) }
func Forbidden(code, message string) *Error  { return New(http.StatusForbidden, code, message) }
func Unauthorized(code, message string) *Error {
	return New(http.StatusUnauthorized, code, message)
}

// As extracts an *Error from err, if any.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// Is reports whether err is (or wraps) the given application error.
func Is(err error, target *Error) bool {
	appErr, ok := As(err)
	return ok && appErr == target
}
