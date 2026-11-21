package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Identity is the verified content of an access token.
type Identity struct {
	UserID         string
	OrganizationID string
	SessionID      string
	ExpiresAt      time.Time
}

// SessionValidator reports whether the session behind an access token is still
// active. It lets logout, password reset and account deletion take effect
// immediately instead of waiting for the access token to expire.
type SessionValidator func(ctx context.Context, sessionID, userID string) (bool, error)

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

	return Identity{
		UserID:         userID,
		OrganizationID: organizationID,
		SessionID:      sessionID,
		ExpiresAt:      expiresAt.Time,
	}, nil
}

// RequireUser authenticates the request with a bearer access token. When a
// session validator is provided the backing session must still be active.
func RequireUser(secret []byte, validators ...SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			httpx.AbortError(c, http.StatusUnauthorized, "missing_token", "missing token")
			return
		}

		identity, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			httpx.AbortError(c, http.StatusUnauthorized, "invalid_token", "invalid token")
			return
		}

		for _, validate := range validators {
			if validate == nil {
				continue
			}
			active, err := validate(c.Request.Context(), identity.SessionID, identity.UserID)
			if err != nil {
				httpx.AbortError(c, http.StatusServiceUnavailable, "session_check_failed", "could not verify session")
				return
			}
			if !active {
				httpx.AbortError(c, http.StatusUnauthorized, "invalid_token", "invalid token")
				return
			}
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
