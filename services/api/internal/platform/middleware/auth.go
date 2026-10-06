package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Identity is the authenticated caller extracted from an access token.
type Identity struct {
	UserID         string
	OrganizationID string
	SessionID      string
}

// SessionValidator confirms the session behind a token is still usable
// (not revoked, not expired, account active) so logout, account deletion and
// suspension take effect immediately instead of when the token expires.
type SessionValidator interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

var errInvalidToken = errors.New("invalid token")

// ParseAccessToken strictly validates an HS256 access token and its claims.
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
		return Identity{}, errInvalidToken
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Identity{}, errInvalidToken
	}
	organizationID, ok := stringClaim(claims, "oid")
	if !ok {
		return Identity{}, errInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Identity{}, errInvalidToken
	}
	return Identity{UserID: userID, OrganizationID: organizationID, SessionID: sessionID}, nil
}

// Authenticate verifies the token and, when a validator is given, the session.
func Authenticate(ctx context.Context, secret []byte, validator SessionValidator, tokenString string) (Identity, error) {
	identity, err := ParseAccessToken(secret, tokenString)
	if err != nil {
		return Identity{}, err
	}
	if validator != nil {
		active, err := validator.IsSessionActive(ctx, identity.SessionID, identity.UserID)
		if err != nil {
			return Identity{}, err
		}
		if !active {
			return Identity{}, errInvalidToken
		}
	}
	return identity, nil
}

func RequireUser(secret []byte, validator SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		identity, err := Authenticate(c.Request.Context(), secret, validator, tokenString)
		if err != nil {
			if errors.Is(err, errInvalidToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not verify session"})
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
