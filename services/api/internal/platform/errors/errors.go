package platformerrors

import (
	"fmt"
	"net/http"
)

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// AppError is an expected, client-facing failure. Services return it for
// validation, permission and state errors; anything else is treated as an
// internal error by the HTTP layer and never leaked to clients.
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func Validation(message string) *AppError {
	return New(http.StatusBadRequest, "validation_error", message)
}

func NotFound(message string) *AppError {
	return New(http.StatusNotFound, "not_found", message)
}

func Conflict(code, message string) *AppError {
	return New(http.StatusConflict, code, message)
}

func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, "forbidden", message)
}
