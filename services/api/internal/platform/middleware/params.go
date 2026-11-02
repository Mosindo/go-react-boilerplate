package middleware

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateIDParams rejects any route parameter named "id" that is not a UUID
// before it reaches SQL, answering 404 like an unknown resource.
func ValidateIDParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		if id := c.Param("id"); id != "" && !uuidRe.MatchString(id) {
			// Routes such as /matches/:id/... always use UUIDs. Notification and
			// photo routes do too, so one rule covers every ":id".
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found", "code": "not_found"})
			return
		}
		c.Next()
	}
}
