// Package httpx holds small helpers shared by feature handlers so every
// endpoint answers errors with the same JSON shape: {"error": "..."}.
package httpx

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// ParseLimit validates a ?limit= query value; empty means def, out of range is rejected.
func ParseLimit(raw string, def, max int) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		return 0, false
	}
	return n, true
}

func IsUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

// ErrorCode adds a stable machine-readable code next to the human message.
func ErrorCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": message, "code": code})
}
