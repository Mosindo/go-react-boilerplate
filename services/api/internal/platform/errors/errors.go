package platformerrors

import (
	"errors"
	"fmt"
)

// Kind classifies an application error so the HTTP layer can map it to a status code
// without services knowing anything about HTTP.
type Kind int

const (
	KindValidation Kind = iota + 1
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindTooLarge
	KindRateLimited
)

// Error is an expected, client-facing failure.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Fields  map[string]string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func Validation(message string, fields map[string]string) *Error {
	return &Error{Kind: KindValidation, Code: "validation_failed", Message: message, Fields: fields}
}

func FieldInvalid(field, reason string) *Error {
	return Validation("Invalid input", map[string]string{field: reason})
}

func Unauthorized(message string) *Error {
	return &Error{Kind: KindUnauthorized, Code: "unauthorized", Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Kind: KindForbidden, Code: "forbidden", Message: message}
}

func NotFound(message string) *Error {
	return &Error{Kind: KindNotFound, Code: "not_found", Message: message}
}

func Conflict(code, message string) *Error {
	return &Error{Kind: KindConflict, Code: code, Message: message}
}

func ProfileIncomplete() *Error {
	return Conflict("profile_incomplete", "Complete your profile (with at least one photo) first")
}

func TooLarge(message string) *Error {
	return &Error{Kind: KindTooLarge, Code: "payload_too_large", Message: message}
}

// As extracts an *Error from err's chain.
func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}
