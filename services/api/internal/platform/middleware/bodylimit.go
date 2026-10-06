package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit caps request bodies so a client cannot make the server buffer
// arbitrarily large JSON. Only POST /photos is exempt: its handler enforces
// its own, larger limit (the Content-Type header is never trusted to relax it).
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		upload := c.Request.Method == http.MethodPost && c.Request.URL.Path == "/photos"
		if !upload && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
