package platformerrors

import (
	"errors"
	"net/http"
)

// AppError is a business error that is safe to show to API clients.
// Services return AppErrors; handlers translate them with httpx.Fail.
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

// Is lets errors.Is match two AppErrors by code, so services can compare
// against package-level sentinels even when handed a copy.
func (e *AppError) Is(target error) bool {
	var other *AppError
	if errors.As(target, &other) {
		return other.Code == e.Code
	}
	return false
}

func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func BadRequest(code, message string) *AppError {
	return New(http.StatusBadRequest, code, message)
}

func Unauthorized(code, message string) *AppError {
	return New(http.StatusUnauthorized, code, message)
}

func Forbidden(code, message string) *AppError {
	return New(http.StatusForbidden, code, message)
}

func NotFound(code, message string) *AppError {
	return New(http.StatusNotFound, code, message)
}

func Conflict(code, message string) *AppError {
	return New(http.StatusConflict, code, message)
}

func TooManyRequests(code, message string) *AppError {
	return New(http.StatusTooManyRequests, code, message)
}

// As extracts an AppError from an error chain.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
