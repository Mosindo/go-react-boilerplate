package middleware

import (
	"context"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/token"
	"github.com/gin-gonic/gin"
)

// SessionChecker reports whether the session referenced by an access token is
// still active. It lets logout, password changes and account deletion revoke
// access immediately instead of waiting for the access token to expire.
type SessionChecker interface {
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
}

func RequireUser(tokens *token.Manager, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpx.ErrorBody{Error: "missing token", Code: "unauthorized"})
			return
		}

		userID, sessionID, err := tokens.Parse(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpx.ErrorBody{Error: "invalid token", Code: "invalid_token"})
			return
		}

		active, err := sessions.IsSessionActive(c.Request.Context(), userID, sessionID)
		if err != nil {
			logger.LogHandlerError(c, "auth.session_check", http.StatusInternalServerError, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, httpx.ErrorBody{Error: "internal server error", Code: "internal_error"})
			return
		}
		if !active {
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpx.ErrorBody{Error: "session expired", Code: "invalid_token"})
			return
		}

		httpx.SetUserID(c, userID)
		c.Set("sessionID", sessionID)
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
