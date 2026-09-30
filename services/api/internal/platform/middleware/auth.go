package middleware

import (
	"context"
	"net/http"
	"strings"

	"example.com/api/internal/platform/authtoken"
	"github.com/gin-gonic/gin"
)

// SessionChecker reports whether a session is still active (not revoked).
// It lets logout, password reset and account deletion take effect before the
// access token expires.
type SessionChecker interface {
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

func RequireUser(tokens *authtoken.Manager, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "unauthorized"})
			return
		}

		claims, err := tokens.Parse(tokenString, authtoken.TypeAccess)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "unauthorized"})
			return
		}

		if sessions != nil {
			active, err := sessions.SessionActive(c.Request.Context(), claims.SessionID, claims.UserID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not verify session", "code": "unavailable"})
				return
			}
			if !active {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired", "code": "unauthorized"})
				return
			}
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
