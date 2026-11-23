package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Identity is the authenticated principal extracted from an access token.
type Identity struct {
	UserID         string
	OrganizationID string
	SessionID      string
}

// SessionChecker confirms the session behind an access token is still active,
// so logout, password reset and account deletion take effect immediately.
type SessionChecker interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// Authenticator validates HS256 access tokens with strict claim checks.
type Authenticator struct {
	secret   []byte
	sessions SessionChecker
}

func NewAuthenticator(secret []byte, sessions SessionChecker) *Authenticator {
	return &Authenticator{secret: secret, sessions: sessions}
}

// Authenticate parses and validates a raw access token.
func (a *Authenticator) Authenticate(ctx context.Context, tokenString string) (Identity, error) {
	if tokenString == "" {
		return Identity{}, ErrInvalidToken
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		return a.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return Identity{}, ErrInvalidToken
	}

	identity := Identity{}
	var ok bool
	if identity.UserID, ok = stringClaim(claims, "uid"); !ok {
		return Identity{}, ErrInvalidToken
	}
	if identity.OrganizationID, ok = stringClaim(claims, "oid"); !ok {
		return Identity{}, ErrInvalidToken
	}
	if identity.SessionID, ok = stringClaim(claims, "sid"); !ok {
		return Identity{}, ErrInvalidToken
	}

	if a.sessions != nil {
		active, err := a.sessions.IsSessionActive(ctx, identity.SessionID, identity.UserID)
		if err != nil {
			return Identity{}, err
		}
		if !active {
			return Identity{}, ErrInvalidToken
		}
	}
	return identity, nil
}

// RequireUser is the Gin middleware protecting authenticated routes.
func (a *Authenticator) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "unauthorized"})
			return
		}
		identity, err := a.Authenticate(c.Request.Context(), tokenString)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "unauthorized"})
				return
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not verify session", "code": "unavailable"})
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
