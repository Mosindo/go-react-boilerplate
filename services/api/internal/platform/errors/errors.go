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

// AppError is an error that is safe to show to API clients.
type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func BadRequest(code, message string) *AppError { return New(http.StatusBadRequest, code, message) }
func Unauthorized(code, message string) *AppError {
	return New(http.StatusUnauthorized, code, message)
}
func Forbidden(code, message string) *AppError { return New(http.StatusForbidden, code, message) }
func NotFound(code, message string) *AppError  { return New(http.StatusNotFound, code, message) }
func Conflict(code, message string) *AppError  { return New(http.StatusConflict, code, message) }

// Validation builds a 400 error carrying per-field messages.
func Validation(fields map[string]string) *AppError {
	return &AppError{
		Status:  http.StatusBadRequest,
		Code:    "validation_failed",
		Message: "some fields are invalid",
		Fields:  fields,
	}
}

// As extracts an *AppError from err's chain.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
