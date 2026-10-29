package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a fixed-window, in-memory limiter. It is per API instance, which
// is enough to blunt brute force and spam; put a shared limiter at the edge when
// running several replicas.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]*rateEntry
	lastGC  time.Time
	now     func() time.Time
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, entries: map[string]*rateEntry{}, now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit.
func (l *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastGC) > l.window {
		for k, e := range l.entries {
			if now.After(e.resetAt) {
				delete(l.entries, k)
			}
		}
		l.lastGC = now
	}

	e, ok := l.entries[key]
	if !ok || now.After(e.resetAt) {
		l.entries[key] = &rateEntry{count: 1, resetAt: now.Add(l.window)}
		return true, 0
	}
	if e.count >= l.limit {
		return false, e.resetAt.Sub(now)
	}
	e.count++
	return true, 0
}

// ByIP limits requests per client IP. A nil limiter disables limiting.
func (l *RateLimiter) ByIP() gin.HandlerFunc {
	return l.handler(func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// ByUser limits requests per authenticated user (use after RequireUser).
func (l *RateLimiter) ByUser() gin.HandlerFunc {
	return l.handler(func(c *gin.Context) string { return "user:" + c.GetString("userID") })
}

func (l *RateLimiter) handler(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil {
			c.Next()
			return
		}
		ok, retry := l.Allow(key(c))
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please slow down"})
			return
		}
		c.Next()
	}
}
