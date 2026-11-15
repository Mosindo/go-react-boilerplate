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

// AccessClaims are the identity claims carried by an access token.
type AccessClaims struct {
	UserID    string
	SessionID string
}

// SessionChecker confirms that the session behind an access token is still
// active. This makes logout, password reset and account deletion take effect
// immediately instead of after the access token TTL.
type SessionChecker interface {
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// ParseAccessToken strictly validates an HS256 access token: algorithm,
// signature, mandatory expiration and required claims.
func ParseAccessToken(secret []byte, tokenString string) (AccessClaims, error) {
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
		return AccessClaims{}, ErrInvalidToken
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return AccessClaims{}, ErrInvalidToken
	}
	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return AccessClaims{}, ErrInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return AccessClaims{}, ErrInvalidToken
	}
	return AccessClaims{UserID: userID, SessionID: sessionID}, nil
}

// Authenticate validates a raw token and checks its session.
func Authenticate(ctx context.Context, secret []byte, sessions SessionChecker, tokenString string) (AccessClaims, error) {
	claims, err := ParseAccessToken(secret, tokenString)
	if err != nil {
		return AccessClaims{}, err
	}
	if sessions != nil {
		active, err := sessions.IsSessionActive(ctx, claims.SessionID, claims.UserID)
		if err != nil {
			return AccessClaims{}, err
		}
		if !active {
			return AccessClaims{}, ErrInvalidToken
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
			if errors.Is(err, ErrInvalidToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "unauthorized"})
				return
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication unavailable", "code": "internal"})
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
