// Package platformerrors defines the transport-agnostic error type returned by services.
package platformerrors

import (
	"errors"
	"fmt"
)

// Kind classifies an error so transports can map it to a status code.
type Kind int

const (
	KindInternal Kind = iota
	KindInvalid
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindTooManyRequests
)

// Error is a business error carrying a stable machine code and a user-facing message.
type Error struct {
	Kind    Kind
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func Invalid(code, message string) *Error      { return New(KindInvalid, code, message) }
func Unauthorized(code, message string) *Error { return New(KindUnauthorized, code, message) }
func Forbidden(code, message string) *Error    { return New(KindForbidden, code, message) }
func NotFound(code, message string) *Error     { return New(KindNotFound, code, message) }
func Conflict(code, message string) *Error     { return New(KindConflict, code, message) }

// As extracts a business error from err, if any.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
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
