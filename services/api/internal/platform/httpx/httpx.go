// Package httpx holds tiny helpers shared by feature handlers.
package httpx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Error writes the API's uniform error envelope: {"error": "<message>"}.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// Limit parses a ?limit= style query value, clamped to [1, max].
func Limit(raw string, def, max int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// Offset parses a non-negative ?offset= value.
func Offset(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func NoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}

func Internal(c *gin.Context) {
	Error(c, http.StatusInternalServerError, "internal error")
}

// Coded writes the error envelope with a stable machine-readable code the
// client can branch on: {"error": "<message>", "code": "<code>"}.
func Coded(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": message, "code": code})
}
