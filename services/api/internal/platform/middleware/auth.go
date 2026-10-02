package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"example.com/api/internal/platform/token"
	"github.com/gin-gonic/gin"
)

// SessionChecker tells whether a session is still active (not revoked, not expired, user exists).
type SessionChecker interface {
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
}

// CachedSessions memoizes positive checks for a short time so that authenticated
// requests do not each cost a DB round trip. Revocation takes effect within ttl
// (immediately on this instance through Forget).
type cacheEntry struct {
	userID string
	exp    time.Time
}

type CachedSessions struct {
	inner SessionChecker
	ttl   time.Duration
	mu    sync.Mutex
	seen  map[string]cacheEntry
}

func NewCachedSessions(inner SessionChecker, ttl time.Duration) *CachedSessions {
	return &CachedSessions{inner: inner, ttl: ttl, seen: make(map[string]cacheEntry)}
}

func (s *CachedSessions) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	s.mu.Lock()
	if e, ok := s.seen[sessionID]; ok && time.Now().Before(e.exp) {
		s.mu.Unlock()
		return true, nil
	}
	s.mu.Unlock()

	ok, err := s.inner.SessionActive(ctx, sessionID, userID)
	if err != nil || !ok {
		return false, err
	}
	s.mu.Lock()
	if len(s.seen) > 50000 {
		s.seen = make(map[string]cacheEntry)
	}
	s.seen[sessionID] = cacheEntry{userID: userID, exp: time.Now().Add(s.ttl)}
	s.mu.Unlock()
	return true, nil
}

func (s *CachedSessions) ForgetSession(sessionID string) {
	s.mu.Lock()
	delete(s.seen, sessionID)
	s.mu.Unlock()
}

// ForgetUser drops every cached session of a user (password change, account deletion).
func (s *CachedSessions) ForgetUser(userID string) {
	s.mu.Lock()
	for id, e := range s.seen {
		if e.userID == userID {
			delete(s.seen, id)
		}
	}
	s.mu.Unlock()
}

func RequireUser(secret []byte, sessions SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			unauthorized(c, "missing token")
			return
		}
		claims, err := token.Parse(secret, raw, token.TypeAccess)
		if err != nil {
			unauthorized(c, "invalid token")
			return
		}
		active, err := sessions.SessionActive(c.Request.Context(), claims.SessionID, claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "something went wrong", "code": "internal"})
			return
		}
		if !active {
			unauthorized(c, "session ended")
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("sessionID", claims.SessionID)
		c.Next()
	}
}

func unauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg, "code": "unauthorized"})
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
