package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionValidator confirms the session behind an access token is still active and the account
// is usable. It lets logout, password reset, suspension and account deletion take effect
// immediately instead of waiting for the short-lived access token to expire.
type SessionValidator func(ctx context.Context, userID, sessionID string) (bool, error)

// ParseAccessToken validates an HS256 access token and returns its user and session ids.
func ParseAccessToken(secret []byte, tokenString string) (userID, sessionID string, err error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	uid, ok := stringClaim(claims, "uid")
	if !ok {
		return "", "", errors.New("invalid token")
	}
	sid, ok := stringClaim(claims, "sid")
	if !ok {
		return "", "", errors.New("invalid token")
	}
	return uid, sid, nil
}

func RequireUser(secret []byte, validate SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			abortUnauthorized(c)
			return
		}
		userID, sessionID, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			abortUnauthorized(c)
			return
		}
		if validate != nil {
			ok, err := validate(c.Request.Context(), userID, sessionID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong", "code": "internal"})
				return
			}
			if !ok {
				abortUnauthorized(c)
				return
			}
		}
		c.Set("userID", userID)
		c.Set("sessionID", sessionID)
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required", "code": "unauthorized"})
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
