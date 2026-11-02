package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxJSONBody caps request bodies at limit bytes. Multipart uploads are
// excluded: the photo handlers enforce their own, larger limit.
func MaxJSONBody(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/") && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
