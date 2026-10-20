package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Identity is what a valid access token proves.
type Identity struct {
	UserID    string
	SessionID string
}

// ParseAccessToken strictly validates an HS256 access token: signing method,
// mandatory expiration and required claims.
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
	}, jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return Identity{}, errors.New("invalid token")
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Identity{}, errors.New("invalid token")
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Identity{}, errors.New("invalid token")
	}
	return Identity{UserID: userID, SessionID: sessionID}, nil
}

// SessionChecker tells whether a session is still live (not revoked, not
// expired, user not deleted). It makes logout, password reset and account
// deletion take effect immediately instead of when the access token expires.
type SessionChecker interface {
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// Authenticate validates a bearer token and its session; checker may be nil.
func Authenticate(ctx context.Context, secret []byte, checker SessionChecker, tokenString string) (Identity, error) {
	identity, err := ParseAccessToken(secret, tokenString)
	if err != nil {
		return Identity{}, err
	}
	if checker != nil {
		active, err := checker.SessionActive(ctx, identity.SessionID, identity.UserID)
		if err != nil {
			return Identity{}, err
		}
		if !active {
			return Identity{}, errors.New("session is no longer active")
		}
	}
	return identity, nil
}

func RequireUser(secret []byte, checker SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		identity, err := Authenticate(c.Request.Context(), secret, checker, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userID", identity.UserID)
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
