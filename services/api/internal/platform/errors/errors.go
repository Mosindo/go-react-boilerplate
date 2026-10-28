package platformerrors

import (
	"errors"
	"fmt"
	"net/http"
)

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// Error is an expected, client-visible failure. Services return it; handlers render it.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(code, message string) *Error   { return New(http.StatusBadRequest, code, message) }
func Unauthorized(code, message string) *Error { return New(http.StatusUnauthorized, code, message) }
func Forbidden(code, message string) *Error    { return New(http.StatusForbidden, code, message) }
func NotFound(code, message string) *Error     { return New(http.StatusNotFound, code, message) }
func Conflict(code, message string) *Error     { return New(http.StatusConflict, code, message) }

// As extracts an *Error from err's chain.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
