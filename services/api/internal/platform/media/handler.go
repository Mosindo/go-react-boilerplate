package media

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ServeHandler streams a stored image when the signed URL is valid.
func ServeHandler(storage Storage, signer *Signer) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		if !signer.Verify(key, c.Query("exp"), c.Query("sig")) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid or expired media link"})
			return
		}
		f, err := storage.Open(key)
		if err != nil {
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidKey) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "media not found"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not read media"})
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not read media"})
			return
		}

		h := c.Writer.Header()
		h.Set("Content-Type", "image/jpeg")
		h.Set("Cache-Control", "private, max-age=3600")
		h.Set("Cross-Origin-Resource-Policy", "cross-origin")
		h.Set("Content-Security-Policy", "default-src 'none'")
		http.ServeContent(c.Writer, c.Request, "", info.ModTime(), f)
	}
}
