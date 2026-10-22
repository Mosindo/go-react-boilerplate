package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// LimitJSONBody caps non-multipart request bodies. Upload handlers set their own,
// larger limit.
func LimitJSONBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/") && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
