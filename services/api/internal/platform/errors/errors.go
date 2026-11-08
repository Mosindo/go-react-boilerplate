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

// AppError is a business error carrying the HTTP status and a client-safe message.
type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string { return e.Message }

func New(status int, message string) *AppError { return &AppError{Status: status, Message: message} }

var (
	ErrNotFound     = New(http.StatusNotFound, "not found")
	ErrForbidden    = New(http.StatusForbidden, "forbidden")
	ErrUnauthorized = New(http.StatusUnauthorized, "unauthorized")
)

func Invalid(message string) *AppError  { return New(http.StatusBadRequest, message) }
func Conflict(message string) *AppError { return New(http.StatusConflict, message) }

// Status maps any error to (status, client message). Unknown errors become opaque 500s.
func Status(err error) (int, string) {
	var app *AppError
	if errors.As(err, &app) {
		return app.Status, app.Message
	}
	return http.StatusInternalServerError, "internal error"
}
