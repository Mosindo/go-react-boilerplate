package middleware

import (
	"context"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/tokens"
	"github.com/gin-gonic/gin"
)

// SessionChecker confirms that the session behind an access token is still
// active (not revoked, not expired, user not deleted). This makes logout,
// password reset and account deletion effective immediately.
type SessionChecker interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

func RequireUser(tm *tokens.Manager, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			httpx.Fail(c, http.StatusUnauthorized, "missing_token", "Authentification requise.", nil)
			return
		}
		claims, err := tm.Parse(raw, tokens.TypeAccess)
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_token", "Session invalide ou expirée.", nil)
			return
		}
		active, err := sessions.IsSessionActive(c.Request.Context(), claims.SessionID, claims.UserID)
		if err != nil {
			httpx.Internal(c, err)
			return
		}
		if !active {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_token", "Session invalide ou expirée.", nil)
			return
		}

		c.Set(httpx.ContextUserID, claims.UserID)
		c.Set(httpx.ContextSessionID, claims.SessionID)
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

// BodyLimit caps request bodies: multipart uploads get uploadMax bytes, every
// other request jsonMax bytes.
func BodyLimit(jsonMax, uploadMax int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			limit := jsonMax
			if strings.HasPrefix(c.ContentType(), "multipart/") {
				limit = uploadMax
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
