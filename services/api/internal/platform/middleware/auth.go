package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ParseAccessToken validates an HS256 access token and returns the user and session ids.
func ParseAccessToken(secret []byte, tokenString string) (userID, sessionID string, err error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return "", "", errors.New("invalid token")
	}
	sessionID, ok = stringClaim(claims, "sid")
	if !ok {
		return "", "", errors.New("invalid token")
	}
	return userID, sessionID, nil
}

// SessionValidator reports whether a session is still usable (not revoked or expired, account active).
type SessionValidator func(ctx context.Context, sessionID string) (bool, error)

// CachedValidator memoises positive and negative answers for ttl, so a hot API does not hit the
// database on every request while logout, password change and account deletion still take effect
// within ttl. A zero ttl disables caching.
func CachedValidator(validate SessionValidator, ttl time.Duration) SessionValidator {
	type entry struct {
		ok    bool
		until time.Time
	}
	var mu sync.Mutex
	cache := map[string]entry{}
	return func(ctx context.Context, sessionID string) (bool, error) {
		now := time.Now()
		if ttl > 0 {
			mu.Lock()
			e, hit := cache[sessionID]
			mu.Unlock()
			if hit && now.Before(e.until) {
				return e.ok, nil
			}
		}
		ok, err := validate(ctx, sessionID)
		if err != nil {
			return false, err
		}
		if ttl > 0 {
			mu.Lock()
			if len(cache) > 10_000 {
				cache = map[string]entry{}
			}
			cache[sessionID] = entry{ok: ok, until: now.Add(ttl)}
			mu.Unlock()
		}
		return ok, nil
	}
}

// RequireUser authenticates the bearer token and checks that its session is still active.
func RequireUser(secret []byte, validate SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "unauthorized"})
			return
		}
		userID, sessionID, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "unauthorized"})
			return
		}
		active, err := validate(c.Request.Context(), sessionID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "temporarily unavailable", "code": "unavailable"})
			return
		}
		if !active {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired", "code": "unauthorized"})
			return
		}
		c.Set("userID", userID)
		c.Set("sessionID", sessionID)
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
