// Package httpx holds the small helpers shared by every feature handler.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func BadRequest(c *gin.Context, message string) { Error(c, http.StatusBadRequest, message) }

func Internal(c *gin.Context) { Error(c, http.StatusInternalServerError, "something went wrong") }

// NoStore marks a response containing personal data as uncacheable by shared caches.
func NoStore(c *gin.Context) { c.Header("Cache-Control", "no-store") }

// IsUUID reports whether s is a canonical 36-character UUID. Handlers use it to
// reject malformed path parameters before they reach the database.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'):
		default:
			return false
		}
	}
	return true
}
