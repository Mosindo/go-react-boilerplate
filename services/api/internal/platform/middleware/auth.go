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

// Claims are the validated identity claims carried by an access token.
type Claims struct {
	UserID    string
	SessionID string
	ExpiresAt time.Time
}

var ErrInvalidToken = errors.New("invalid token")

// SessionChecker tells whether a login session is still usable (not revoked, not expired, user still exists).
// Checking on every request makes logout, password reset and account deletion effective immediately.
type SessionChecker interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// ParseAccessToken strictly validates an HS256 access token: algorithm, signature, mandatory expiry and claims.
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
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Claims{}, ErrInvalidToken
	}
	return Claims{UserID: userID, SessionID: sessionID, ExpiresAt: expiresAt.Time}, nil
}

// Authenticate validates a token and its session; used by REST middleware and the WebSocket handshake.
func Authenticate(ctx context.Context, secret []byte, sessions SessionChecker, tokenString string) (Claims, error) {
	claims, err := ParseAccessToken(secret, tokenString)
	if err != nil {
		return Claims{}, err
	}
	if sessions != nil {
		active, err := sessions.IsSessionActive(ctx, claims.SessionID, claims.UserID)
		if err != nil || !active {
			return Claims{}, ErrInvalidToken
		}
	}
	return claims, nil
}

func RequireUser(secret []byte, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "unauthorized"})
			return
		}
		claims, err := Authenticate(c.Request.Context(), secret, sessions, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "unauthorized"})
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
