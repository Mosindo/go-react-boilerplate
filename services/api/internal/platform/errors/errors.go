package platformerrors

import (
	"errors"
	"fmt"
)

// Kinds of application errors. Services return *AppError values wrapping one of
// these so the HTTP layer can map them to status codes without string matching.
var (
	ErrInvalid      = errors.New("invalid")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrRateLimited  = errors.New("rate limited")
)

// AppError is an error with a client-safe message.
type AppError struct {
	Kind    error
	Message string
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Kind }

func newError(kind error, msg string, args ...any) error {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return &AppError{Kind: kind, Message: msg}
}

func Invalid(msg string, args ...any) error      { return newError(ErrInvalid, msg, args...) }
func Unauthorized(msg string, args ...any) error { return newError(ErrUnauthorized, msg, args...) }
func Forbidden(msg string, args ...any) error    { return newError(ErrForbidden, msg, args...) }
func NotFound(msg string, args ...any) error     { return newError(ErrNotFound, msg, args...) }
func Conflict(msg string, args ...any) error     { return newError(ErrConflict, msg, args...) }

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
