// Package httpx holds small Gin helpers shared by every feature handler.
package httpx

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Error writes the API-wide error envelope: {"error": "<message>"}.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// UserID returns the authenticated user id set by middleware.RequireUser.
func UserID(c *gin.Context) string {
	return c.GetString("userID")
}

// Limit parses a ?limit= query value, falling back to def and capped at max.
func Limit(raw string, def, max int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return def
	}
	if value > max {
		return max
	}
	return value
}

// Offset parses a ?offset= query value, never negative.
func Offset(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// NoStore marks a response as non-cacheable (private data).
func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}

// Status shortcut for 204 responses.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// ValidationError is returned by services for input the client can fix.
// Handlers answer it with 400 and the message.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func Invalid(message string) error { return &ValidationError{Message: message} }

// AsValidation reports whether err is a client-fixable ValidationError.
func AsValidation(err error) (*ValidationError, bool) {
	var v *ValidationError
	if errors.As(err, &v) {
		return v, true
	}
	return nil, false
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical UUID (guards Postgres uuid casts).
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }

// ParamUUID reads a path parameter and answers 404 when it is not a UUID.
func ParamUUID(c *gin.Context, name string) (string, bool) {
	v := c.Param(name)
	if !IsUUID(v) {
		Error(c, http.StatusNotFound, "not found")
		return "", false
	}
	return strings.ToLower(v), true
}
