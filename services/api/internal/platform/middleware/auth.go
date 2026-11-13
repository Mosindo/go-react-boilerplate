package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims are the validated identity carried by an access token.
type Claims struct {
	UserID         string
	OrganizationID string
	SessionID      string
	ExpiresAt      time.Time
}

// SessionValidator confirms that the session behind a token is still active
// (not revoked by logout, password reset or account deletion).
type SessionValidator interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

var ErrInvalidToken = errors.New("invalid token")

// ParseAccessToken strictly validates an HS256 access token.
func ParseAccessToken(secret []byte, tokenString string) (Claims, error) {
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
		return Claims{}, ErrInvalidToken
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return Claims{}, ErrInvalidToken
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Claims{}, ErrInvalidToken
	}
	organizationID, ok := stringClaim(claims, "oid")
	if !ok {
		return Claims{}, ErrInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Claims{}, ErrInvalidToken
	}

	return Claims{
		UserID:         userID,
		OrganizationID: organizationID,
		SessionID:      sessionID,
		ExpiresAt:      expiresAt.Time,
	}, nil
}

// RequireUser authenticates the request with a bearer access token and, when
// a SessionValidator is provided, checks the session is still active.
func RequireUser(secret []byte, sessions SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "unauthenticated"})
			return
		}

		claims, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "unauthenticated"})
			return
		}

		if sessions != nil {
			active, err := sessions.IsSessionActive(c.Request.Context(), claims.SessionID, claims.UserID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not validate session"})
				return
			}
			if !active {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired", "code": "unauthenticated"})
				return
			}
		}

		c.Set("userID", claims.UserID)
		c.Set("organizationID", claims.OrganizationID)
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
