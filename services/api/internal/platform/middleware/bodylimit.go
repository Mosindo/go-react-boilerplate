package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BodyLimit caps request bodies. Multipart uploads are exempt: their handlers
// enforce their own (larger) limit.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/") {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
