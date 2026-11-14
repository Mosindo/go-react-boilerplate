package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionChecker confirms that the session referenced by an access token is
// still active, so logout, password reset and account deletion revoke
// access immediately instead of waiting for token expiry.
type SessionChecker func(ctx context.Context, sessionID, userID string) (bool, error)

// Identity is the authenticated principal extracted from an access token.
type Identity struct {
	UserID    string
	SessionID string
}

var errInvalidToken = errors.New("invalid token")

// ParseAccessToken strictly validates an HS256 access token and its claims.
func ParseAccessToken(secret []byte, tokenString string) (Identity, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid {
		return Identity{}, errInvalidToken
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Identity{}, errInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Identity{}, errInvalidToken
	}
	return Identity{UserID: userID, SessionID: sessionID}, nil
}

// Authenticate validates a raw access token and its session.
func Authenticate(ctx context.Context, secret []byte, checker SessionChecker, tokenString string) (Identity, error) {
	identity, err := ParseAccessToken(secret, tokenString)
	if err != nil {
		return Identity{}, err
	}
	if checker != nil {
		active, err := checker(ctx, identity.SessionID, identity.UserID)
		if err != nil {
			return Identity{}, err
		}
		if !active {
			return Identity{}, errInvalidToken
		}
	}
	return identity, nil
}

func RequireUser(secret []byte, checker SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentification requise", "code": "missing_token"})
			return
		}

		identity, err := Authenticate(c.Request.Context(), secret, checker, tokenString)
		if err != nil {
			if errors.Is(err, errInvalidToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session invalide ou expirée", "code": "invalid_token"})
				return
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "service indisponible", "code": "unavailable"})
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
