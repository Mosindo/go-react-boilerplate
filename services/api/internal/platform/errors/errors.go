// Package platformerrors defines the application error type shared by services and handlers.
package platformerrors

import (
	"errors"
	"fmt"
)

// Error is a business error carrying an HTTP-friendly status and a stable machine code.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

// New builds an application error.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// As extracts an application error from err, if any.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// Wrap adds operation context to err.
func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
