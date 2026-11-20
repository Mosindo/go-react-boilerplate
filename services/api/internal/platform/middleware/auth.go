package middleware

import (
	"context"
	"net/http"
	"strings"

	"example.com/api/internal/platform/authtoken"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// SessionChecker confirms that the session behind a token is still active
// (not logged out, not expired, account not suspended or deleted).
type SessionChecker interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// RequireUser authenticates a request with a strict HS256 access token and
// verifies server-side that the session has not been revoked.
func RequireUser(secret []byte, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			abortUnauthorized(c, "missing token")
			return
		}

		claims, err := authtoken.Parse(secret, tokenString, authtoken.TypeAccess)
		if err != nil {
			abortUnauthorized(c, "invalid token")
			return
		}

		active, err := sessions.IsSessionActive(c.Request.Context(), claims.SessionID, claims.UserID)
		if err != nil {
			logger.LogHandlerError(c, "auth.session_check", http.StatusServiceUnavailable, err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication temporarily unavailable", "code": "auth_unavailable"})
			return
		}
		if !active {
			abortUnauthorized(c, "session expired")
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("sessionID", claims.SessionID)
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message, "code": "unauthorized"})
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
