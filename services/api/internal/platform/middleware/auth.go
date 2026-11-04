package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionChecker reports whether a session is still active (not revoked, not expired).
type SessionChecker interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// ParseAccessToken validates an HS256 access token and returns (userID, sessionID).
func ParseAccessToken(secret []byte, tokenString string) (string, string, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return "", "", errors.New("invalid token")
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return "", "", errors.New("invalid token")
	}
	return userID, sessionID, nil
}

// RequireUser authenticates the request and checks the session is still active.
func RequireUser(secret []byte, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "unauthorized"})
			return
		}
		userID, sessionID, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "unauthorized"})
			return
		}
		active, err := sessions.IsSessionActive(c.Request.Context(), sessionID, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal"})
			return
		}
		if !active {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired", "code": "unauthorized"})
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

func stringClaim(claims jwt.MapClaims, key string) (string, bool) {
	value, ok := claims[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}
