package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionChecker reports whether a session is still valid (not revoked/expired
// and its owner still exists). It lets logout and account deletion take effect
// immediately instead of waiting for the access token to expire.
type SessionChecker interface {
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

func RequireUser(secret []byte, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			unauthorized(c, "missing token")
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method type")
			}
			return secret, nil
		}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			unauthorized(c, "invalid token")
			return
		}

		userID, ok := stringClaim(claims, "uid")
		if !ok {
			unauthorized(c, "invalid token")
			return
		}
		sessionID, ok := stringClaim(claims, "sid")
		if !ok {
			unauthorized(c, "invalid token")
			return
		}

		if sessions != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			active, err := sessions.SessionActive(ctx, sessionID, userID)
			cancel()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "service unavailable", "code": "unavailable"})
				return
			}
			if !active {
				unauthorized(c, "session expired")
				return
			}
		}

		c.Set("userID", userID)
		c.Set("sessionID", sessionID)
		c.Next()
	}
}

func unauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message, "code": "unauthorized"})
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func stringClaim(claims jwt.MapClaims, key string) (string, bool) {
	value, ok := claims[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}
