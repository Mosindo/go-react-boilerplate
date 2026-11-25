// Package apperr defines typed application errors that carry an HTTP status,
// a stable machine-readable code and a user-facing message. Services return
// these errors; handlers translate them with httpx.Fail.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func Validation(code, message string) *Error {
	return New(http.StatusBadRequest, code, message)
}

var (
	ErrUnauthorized = New(http.StatusUnauthorized, "unauthorized", "authentication required")
	ErrForbidden    = New(http.StatusForbidden, "forbidden", "you are not allowed to perform this action")
	ErrNotFound     = New(http.StatusNotFound, "not_found", "resource not found")
	ErrInvalidInput = New(http.StatusBadRequest, "invalid_request", "invalid request")
	ErrRateLimited  = New(http.StatusTooManyRequests, "rate_limited", "too many requests, please slow down")
)

// As returns the typed application error contained in err, if any.
func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// Wrap annotates an unexpected error with the failing operation.
func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
