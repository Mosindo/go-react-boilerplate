package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionValidator confirms that the session referenced by an access token is still active.
type SessionValidator interface {
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

func unauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message, "code": "unauthorized"})
}

// RequireUser validates an HS256 access token (signature, algorithm, mandatory expiry,
// required claims) and then checks that the session was not revoked.
func RequireUser(secret []byte, sessions SessionValidator) gin.HandlerFunc {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)

	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			unauthorized(c, "missing token")
			return
		}

		userID, sessionID, err := parseAccessToken(parser, secret, tokenString)
		if err != nil {
			unauthorized(c, "invalid token")
			return
		}

		if sessions != nil {
			active, err := sessions.SessionActive(c.Request.Context(), sessionID, userID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication temporarily unavailable", "code": "unavailable"})
				return
			}
			if !active {
				unauthorized(c, "session revoked")
				return
			}
		}

		c.Set("userID", userID)
		c.Set("sessionID", sessionID)
		c.Next()
	}
}

func parseAccessToken(parser *jwt.Parser, secret []byte, tokenString string) (string, string, error) {
	claims := jwt.MapClaims{}
	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return "", "", errors.New("missing uid")
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return "", "", errors.New("missing sid")
	}
	return userID, sessionID, nil
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
