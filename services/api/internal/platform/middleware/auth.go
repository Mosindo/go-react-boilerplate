package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Identity is the authenticated principal extracted from an access token.
type Identity struct {
	UserID         string
	OrganizationID string
	SessionID      string
}

// SessionChecker confirms that the session behind an access token is still
// active, so logout, password reset and account deletion take effect
// immediately instead of when the short-lived JWT expires.
type SessionChecker interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

var ErrInvalidToken = errors.New("invalid token")

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
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return Identity{}, ErrInvalidToken
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return Identity{}, ErrInvalidToken
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Identity{}, ErrInvalidToken
	}
	organizationID, ok := stringClaim(claims, "oid")
	if !ok {
		return Identity{}, ErrInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Identity{}, ErrInvalidToken
	}

	return Identity{UserID: userID, OrganizationID: organizationID, SessionID: sessionID}, nil
}

// Authenticate validates a raw token and, when a checker is configured, the
// underlying session.
func Authenticate(ctx context.Context, secret []byte, sessions SessionChecker, tokenString string) (Identity, error) {
	identity, err := ParseAccessToken(secret, tokenString)
	if err != nil {
		return Identity{}, err
	}
	if sessions != nil {
		active, err := sessions.IsSessionActive(ctx, identity.SessionID, identity.UserID)
		if err != nil {
			return Identity{}, err
		}
		if !active {
			return Identity{}, ErrInvalidToken
		}
	}
	return identity, nil
}

// RequireUser rejects requests without a valid bearer token and stores the
// identity in the Gin context.
func RequireUser(secret []byte, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentification requise.", "code": "missing_token"})
			return
		}

		identity, err := Authenticate(c.Request.Context(), secret, sessions, tokenString)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session invalide ou expirée.", "code": "invalid_token"})
				return
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Service momentanément indisponible.", "code": "unavailable"})
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
