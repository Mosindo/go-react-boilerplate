package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Identity is the authenticated caller extracted from a valid access token.
type Identity struct {
	UserID         string
	OrganizationID string
	SessionID      string
}

// ParseAccessToken validates an HS256 access token and returns the caller identity.
func ParseAccessToken(secret []byte, tokenString string) (Identity, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		return secret, nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return Identity{}, errors.New("invalid token")
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Identity{}, errors.New("invalid token")
	}
	organizationID, ok := stringClaim(claims, "oid")
	if !ok {
		return Identity{}, errors.New("invalid token")
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Identity{}, errors.New("invalid token")
	}
	return Identity{UserID: userID, OrganizationID: organizationID, SessionID: sessionID}, nil
}

func RequireUser(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		identity, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("userID", identity.UserID)
		c.Set("organizationID", identity.OrganizationID)
		c.Set("sessionID", identity.SessionID)
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
