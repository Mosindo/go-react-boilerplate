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

// AppError is a business error that is safe to show to API clients.
// Services return it; handlers translate it through httpx.Respond.
type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string { return e.Message }

func newAppError(status int, message string) *AppError {
	return &AppError{Status: status, Message: message}
}

func BadRequest(message string) *AppError   { return newAppError(http.StatusBadRequest, message) }
func Unauthorized(message string) *AppError { return newAppError(http.StatusUnauthorized, message) }
func Forbidden(message string) *AppError    { return newAppError(http.StatusForbidden, message) }
func NotFound(message string) *AppError     { return newAppError(http.StatusNotFound, message) }
func Conflict(message string) *AppError     { return newAppError(http.StatusConflict, message) }

// AsAppError extracts an AppError from err's chain.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
