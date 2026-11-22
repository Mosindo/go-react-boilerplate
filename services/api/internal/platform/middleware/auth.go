package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionValidator reports whether a session is still active (not revoked,
// not expired). It lets logout, password changes and account deletion take
// effect immediately instead of waiting for access-token expiry.
type SessionValidator func(ctx context.Context, sessionID, userID string) (bool, error)

// RequireUser validates the bearer token only (stateless).
func RequireUser(secret []byte) gin.HandlerFunc {
	return RequireActiveSession(secret, nil)
}

// RequireActiveSession validates the bearer token and, when sessions is not
// nil, checks the session against the database.
func RequireActiveSession(secret []byte, sessions SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
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
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		expiresAt, err := claims.GetExpirationTime()
		if err != nil || expiresAt == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		userID, ok := stringClaim(claims, "uid")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		organizationID, ok := stringClaim(claims, "oid")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		sessionID, ok := stringClaim(claims, "sid")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		if sessions != nil {
			active, err := sessions(c.Request.Context(), sessionID, userID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not validate session"})
				return
			}
			if !active {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired", "code": "session_revoked"})
				return
			}
		}

		c.Set("userID", userID)
		c.Set("organizationID", organizationID)
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

func stringClaim(claims jwt.MapClaims, key string) (string, bool) {
	value, ok := claims[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}
