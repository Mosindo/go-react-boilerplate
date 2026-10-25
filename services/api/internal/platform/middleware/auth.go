package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionChecker reports whether a session is still active (not revoked/expired, user not deleted).
type SessionChecker interface {
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
}

// Identity is the validated content of an access token.
type Identity struct {
	UserID    string
	SessionID string
}

var ErrInvalidToken = errors.New("invalid token")

// ParseAccessToken strictly validates an HS256 access token: algorithm, expiration and required claims.
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
		return Identity{}, ErrInvalidToken
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Identity{}, ErrInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Identity{}, ErrInvalidToken
	}
	return Identity{UserID: userID, SessionID: sessionID}, nil
}

// Authenticate validates a bearer token and its session.
func Authenticate(ctx context.Context, secret []byte, checker SessionChecker, tokenString string) (Identity, bool) {
	identity, err := ParseAccessToken(secret, tokenString)
	if err != nil {
		return Identity{}, false
	}
	active, err := checker.IsSessionActive(ctx, identity.UserID, identity.SessionID)
	if err != nil || !active {
		return Identity{}, false
	}
	return identity, true
}

// RequireUser authenticates the request and applies a general per-user budget (limiter may be nil).
func RequireUser(secret []byte, checker SessionChecker, limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "unauthorized"})
			return
		}
		identity, ok := Authenticate(c.Request.Context(), secret, checker, tokenString)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "unauthorized"})
			return
		}
		c.Set("userID", identity.UserID)
		c.Set("sessionID", identity.SessionID)
		if limiter != nil {
			if ok, wait := limiter.Allow("api:" + identity.UserID); !ok {
				c.Header("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
				return
			}
		}
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
