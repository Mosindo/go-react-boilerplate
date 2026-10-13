package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TokenIssuer is the `iss` claim of every access token this API signs.
const TokenIssuer = "go-react-saas"

// UserClaims are the identity claims extracted from a validated access token.
type UserClaims struct {
	UserID    string
	SessionID string
}

// ParseAccessToken strictly validates an HS256 access token: signing method,
// mandatory expiry, issuer and the identity claims.
func ParseAccessToken(secret []byte, tokenString string) (UserClaims, bool) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(TokenIssuer),
	)
	if err != nil || !token.Valid {
		return UserClaims{}, false
	}
	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return UserClaims{}, false
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return UserClaims{}, false
	}
	return UserClaims{UserID: userID, SessionID: sessionID}, true
}

func RequireUser(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, ok := ParseAccessToken(secret, tokenString)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
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
