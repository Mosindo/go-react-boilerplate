package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims are the validated identity carried by an access token.
type Claims struct {
	UserID    string
	SessionID string
}

var errInvalidToken = errors.New("invalid token")

// ParseAccessToken strictly validates an HS256 access token: algorithm, signature,
// mandatory expiration and the presence of the uid/sid claims.
func ParseAccessToken(secret []byte, tokenString string) (Claims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return Claims{}, errInvalidToken
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Claims{}, errInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Claims{}, errInvalidToken
	}
	return Claims{UserID: userID, SessionID: sessionID}, nil
}

func RequireUser(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "missing_token"})
			return
		}
		claims, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "invalid_token"})
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

func stringClaim(claims jwt.MapClaims, key string) (string, bool) {
	value, ok := claims[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}
