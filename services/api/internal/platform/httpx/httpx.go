// Package httpx holds small helpers shared by Gin handlers.
package httpx

import (
	"regexp"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Fail logs the internal error and answers with a stable, non-leaky message.
func Fail(c *gin.Context, operation string, status int, message string, err error) {
	if err != nil {
		logger.LogHandlerError(c, operation, status, err)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// IsUUID reports whether value is a canonical UUID string.
func IsUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

// UUIDParam reads a path parameter and validates it as a UUID.
func UUIDParam(c *gin.Context, name string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(c.Param(name)))
	if !IsUUID(value) {
		return "", false
	}
	return value, true
}

// Limit parses a "limit" style query value, clamped to [1, max].
func Limit(raw string, fallback, max int) int {
	return boundedInt(raw, fallback, 1, max)
}

// Offset parses an "offset" style query value, clamped to [0, max].
func Offset(raw string, max int) int {
	return boundedInt(raw, 0, 0, max)
}

func boundedInt(raw string, fallback, min, max int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
