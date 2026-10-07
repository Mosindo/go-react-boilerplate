package platformerrors

import (
	"errors"
	"fmt"
	"net/http"
)

// Machine-readable error codes returned to API clients (see docs/API.md).
const (
	CodeValidation        = "validation_error"
	CodeUnauthorized      = "unauthorized"
	CodeForbidden         = "forbidden"
	CodeNotFound          = "not_found"
	CodeConflict          = "conflict"
	CodeAlreadySwiped     = "already_swiped"
	CodeProfileIncomplete = "profile_incomplete"
	CodeUnderage          = "underage"
	CodeBlocked           = "blocked"
	CodeRateLimited       = "rate_limited"
	CodeInternal          = "internal"
)

// Error is an expected, client-facing failure raised by services. Anything that is
// not an *Error is treated as an unexpected internal failure by the HTTP layer.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func Validation(message string) *Error {
	return New(http.StatusBadRequest, CodeValidation, message)
}

func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, CodeUnauthorized, message)
}

func Forbidden(message string) *Error {
	return New(http.StatusForbidden, CodeForbidden, message)
}

func NotFound(message string) *Error {
	return New(http.StatusNotFound, CodeNotFound, message)
}

func Conflict(message string) *Error {
	return New(http.StatusConflict, CodeConflict, message)
}

// As extracts a client-facing error from err, if any.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
