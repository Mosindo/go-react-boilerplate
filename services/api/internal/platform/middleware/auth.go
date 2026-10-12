package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionValidator reports whether the session behind an access token is still
// live (not logged out, not expired, user not disabled or deleted).
type SessionValidator func(ctx context.Context, userID, sessionID string) bool

// ParseAccessToken strictly validates an HS256 access token and returns the
// user and session ids it carries. Expiration is mandatory.
func ParseAccessToken(secret []byte, tokenString string) (userID, sessionID string, err error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		return secret, nil
	}, jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return "", "", errors.New("invalid token")
	}
	sessionID, ok = stringClaim(claims, "sid")
	if !ok {
		return "", "", errors.New("invalid token")
	}
	return userID, sessionID, nil
}

func RequireUser(secret []byte, validate SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		userID, sessionID, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if validate != nil && !validate(c.Request.Context(), userID, sessionID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
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
