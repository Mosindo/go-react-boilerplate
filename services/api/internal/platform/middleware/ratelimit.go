package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type window struct {
	count   int
	resetAt time.Time
}

// Limiter is a fixed-window counter keyed by caller. It is in-memory, so limits
// apply per API instance; put a shared limiter behind a load balancer if you scale out.
type Limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	entries map[string]*window
}

func NewLimiter(limit int, per time.Duration) *Limiter {
	return &Limiter{limit: limit, window: per, now: time.Now, entries: make(map[string]*window)}
}

// Allow records a hit and reports whether it is within the limit, plus the
// time until the window resets.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) > 10_000 {
		for k, w := range l.entries {
			if now.After(w.resetAt) {
				delete(l.entries, k)
			}
		}
	}

	w, ok := l.entries[key]
	if !ok || now.After(w.resetAt) {
		w = &window{resetAt: now.Add(l.window)}
		l.entries[key] = w
	}
	w.count++
	return w.count <= l.limit, w.resetAt.Sub(now)
}

// ByIP limits per client IP.
func (l *Limiter) ByIP() gin.HandlerFunc {
	return l.handler(func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// ByUser limits per authenticated user; it must run after RequireUser.
func (l *Limiter) ByUser() gin.HandlerFunc {
	return l.handler(func(c *gin.Context) string {
		if id := c.GetString("userID"); id != "" {
			return "user:" + id
		}
		return "ip:" + c.ClientIP()
	})
}

func (l *Limiter) handler(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retry := l.Allow(key(c))
		if !ok {
			seconds := int(retry.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please slow down"})
			return
		}
		c.Next()
	}
}
