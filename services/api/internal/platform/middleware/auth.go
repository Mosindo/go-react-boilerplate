package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/tokens"
	"github.com/gin-gonic/gin"
)

// SessionChecker confirms that the session behind an access token is still
// active. It makes logout, password reset and account deletion take effect
// immediately instead of waiting for the access token to expire.
type SessionChecker interface {
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
}

// RequireUser validates the bearer access token and the backing session.
func RequireUser(manager *tokens.Manager, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			httpx.Abort(c, http.StatusUnauthorized, "missing_token", "authentication required")
			return
		}
		claims, err := manager.Parse(raw, tokens.AudienceAccess)
		if err != nil {
			httpx.Abort(c, http.StatusUnauthorized, "invalid_token", "invalid or expired token")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		active, err := sessions.IsSessionActive(ctx, claims.UserID, claims.SessionID)
		cancel()
		if err != nil {
			logger.LogHandlerError(c, "auth.session_check", http.StatusInternalServerError, err)
			httpx.Abort(c, http.StatusInternalServerError, "internal_error", "something went wrong, please retry")
			return
		}
		if !active {
			httpx.Abort(c, http.StatusUnauthorized, "session_revoked", "session expired, please sign in again")
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
