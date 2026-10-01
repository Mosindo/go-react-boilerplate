package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Limiter is an in-memory fixed-window rate limiter. It protects a single API instance;
// run a shared limiter (e.g. at the reverse proxy) when scaling horizontally.
type Limiter struct {
	mu      sync.Mutex
	window  time.Duration
	max     int
	entries map[string]*window
	now     func() time.Time
}

type window struct {
	start time.Time
	count int
}

func NewLimiter(max int, per time.Duration) *Limiter {
	return &Limiter{window: per, max: max, entries: make(map[string]*window), now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit,
// plus the time to wait before retrying when it is not.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.entries) > 10_000 {
		for k, w := range l.entries {
			if now.Sub(w.start) >= l.window {
				delete(l.entries, k)
			}
		}
	}
	w, ok := l.entries[key]
	if !ok || now.Sub(w.start) >= l.window {
		l.entries[key] = &window{start: now, count: 1}
		return true, 0
	}
	if w.count >= l.max {
		return false, l.window - now.Sub(w.start)
	}
	w.count++
	return true, 0
}

// ByIP limits per client IP (use before authentication, e.g. login).
func (l *Limiter) ByIP() gin.HandlerFunc {
	return l.handler(func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// ByUser limits per authenticated user; it must run after RequireUser.
func (l *Limiter) ByUser() gin.HandlerFunc {
	return l.handler(func(c *gin.Context) string { return "user:" + c.GetString("userID") })
}

func (l *Limiter) handler(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retry := l.Allow(key(c))
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "trop de requêtes, réessayez plus tard", "code": "rate_limited"})
			return
		}
		c.Next()
	}
}
