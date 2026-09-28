package middleware

import (
	"context"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/jwtauth"
	"github.com/gin-gonic/gin"
)

// SessionChecker verifies that a session is still usable: not revoked, not
// expired, and its user still exists.
type SessionChecker interface {
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// RequireUser authenticates the request from a Bearer access token (HS256,
// mandatory exp) and then confirms with the database that the session was not
// revoked and the user has not been deleted. Sets userID and sessionID.
func RequireUser(secret []byte, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			httpx.Abort(c, 401, httpx.CodeUnauthorized, "missing token")
			return
		}
		claims, err := jwtauth.Parse(secret, tokenString)
		if err != nil {
			httpx.Abort(c, 401, httpx.CodeUnauthorized, "invalid token")
			return
		}
		if !httpx.IsUUID(claims.UserID) || !httpx.IsUUID(claims.SessionID) {
			httpx.Abort(c, 401, httpx.CodeUnauthorized, "invalid token")
			return
		}
		userID := strings.ToLower(claims.UserID)
		sessionID := strings.ToLower(claims.SessionID)
		ok, err := sessions.SessionActive(c.Request.Context(), sessionID, userID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if !ok {
			httpx.Abort(c, 401, httpx.CodeUnauthorized, "session is no longer valid")
			return
		}
		c.Set("userID", userID)
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
