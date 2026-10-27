// Package httpx holds the small helpers shared by every HTTP handler so that
// error payloads and pagination parsing stay consistent across features.
package httpx

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

// Error writes the standard error payload: {"error": "<message>"}.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// Limit parses the "limit" query parameter and clamps it to [1, MaxLimit].
func Limit(c *gin.Context, fallback int) int {
	if fallback <= 0 {
		fallback = DefaultLimit
	}
	raw := c.Query("limit")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	if n > MaxLimit {
		return MaxLimit
	}
	return n
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical textual UUID.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }

// UUIDParam reads a path parameter and answers 404 when it is not a UUID, which
// keeps malformed identifiers from ever reaching the database.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	v := c.Param(name)
	if !IsUUID(v) {
		Error(c, http.StatusNotFound, "not found")
		return "", false
	}
	return strings.ToLower(v), true
}
