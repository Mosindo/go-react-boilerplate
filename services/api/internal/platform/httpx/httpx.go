// Package httpx holds small helpers that keep API responses consistent.
package httpx

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// Error writes the standard error envelope: {"error": "message"}.
func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// IntQuery parses an integer query parameter, clamped to [min, max].
func IntQuery(c *gin.Context, key string, def, min, max int) int {
	raw := c.Query(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
