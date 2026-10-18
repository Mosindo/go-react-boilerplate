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

// RateLimiter is a fixed-window in-memory limiter. It is process local, which is
// enough for a single API instance; swap the store for Redis when scaling out.
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
	max     int
	period  time.Duration
	enabled bool
	now     func() time.Time
}

func NewRateLimiter(max int, period time.Duration, enabled bool) *RateLimiter {
	return &RateLimiter{
		windows: make(map[string]*window),
		max:     max,
		period:  period,
		enabled: enabled,
		now:     time.Now,
	}
}

// Allow records one hit for key and reports whether it is within the limit,
// along with the seconds until the window resets.
func (l *RateLimiter) Allow(key string) (bool, int) {
	if !l.enabled {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.windows) > 10_000 {
		for k, w := range l.windows {
			if now.After(w.resetAt) {
				delete(l.windows, k)
			}
		}
	}

	w, ok := l.windows[key]
	if !ok || now.After(w.resetAt) {
		w = &window{resetAt: now.Add(l.period)}
		l.windows[key] = w
	}
	w.count++
	if w.count > l.max {
		retry := int(w.resetAt.Sub(now).Seconds()) + 1
		return false, retry
	}
	return true, 0
}

// ByIP limits per client IP; ByUser limits per authenticated user (falls back to IP).
func (l *RateLimiter) ByIP(scope string) gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string { return scope + ":" + c.ClientIP() })
}

func (l *RateLimiter) ByUser(scope string) gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string {
		if uid := c.GetString("userID"); uid != "" {
			return scope + ":u:" + uid
		}
		return scope + ":" + c.ClientIP()
	})
}

func (l *RateLimiter) middleware(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retry := l.Allow(key(c))
		if !ok {
			c.Header("Retry-After", strconv.Itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please slow down"})
			return
		}
		c.Next()
	}
}
