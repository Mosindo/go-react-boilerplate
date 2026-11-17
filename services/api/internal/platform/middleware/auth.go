package middleware

import (
	"net/http"
	"strings"

	"example.com/api/internal/platform/authtoken"
	"github.com/gin-gonic/gin"
)

// RequireUser authenticates the request with a Bearer access token.
func RequireUser(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentification requise.", "code": "missing_token"})
			return
		}

		claims, err := authtoken.Parse(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session invalide ou expirée.", "code": "invalid_token"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("sessionID", claims.SessionID)
		c.Next()
	}
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
