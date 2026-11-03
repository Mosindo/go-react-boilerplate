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

// AppError is a client-facing business error with a stable code.
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Message }

func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

var (
	ErrNotFound     = New(http.StatusNotFound, "not_found", "not found")
	ErrForbidden    = New(http.StatusForbidden, "forbidden", "forbidden")
	ErrUnauthorized = New(http.StatusUnauthorized, "unauthorized", "unauthorized")
)

func Invalid(message string) *AppError {
	return New(http.StatusBadRequest, "invalid_request", message)
}

func Conflict(code, message string) *AppError {
	return New(http.StatusConflict, code, message)
}

// As extracts an *AppError from err.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
