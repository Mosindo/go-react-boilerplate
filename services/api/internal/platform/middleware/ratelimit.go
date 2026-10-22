package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory fixed-window limiter keyed by an arbitrary string.
// It protects a single API instance; behind several instances use a shared store.
type RateLimiter struct {
	limit  int
	window time.Duration
	mu     sync.Mutex
	hits   map[string]*window
	now    func() time.Time
}

type window struct {
	start time.Time
	count int
}

func NewRateLimiter(limit int, per time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: per, hits: make(map[string]*window), now: time.Now}
}

// Allow records a hit and reports whether the key is still within its budget.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.hits) > 10000 {
		for k, w := range l.hits {
			if now.Sub(w.start) >= l.window {
				delete(l.hits, k)
			}
		}
	}
	w, ok := l.hits[key]
	if !ok || now.Sub(w.start) >= l.window {
		l.hits[key] = &window{start: now, count: 1}
		return true, 0
	}
	if w.count >= l.limit {
		return false, l.window - now.Sub(w.start)
	}
	w.count++
	return true, 0
}

// ByIP limits requests per client IP.
func (l *RateLimiter) ByIP() gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// ByUser limits requests per authenticated user (falls back to IP). Use after RequireUser.
func (l *RateLimiter) ByUser() gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string {
		if id := c.GetString("userID"); id != "" {
			return "user:" + id
		}
		return "ip:" + c.ClientIP()
	})
}

func (l *RateLimiter) middleware(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retry := l.Allow(c.FullPath() + "|" + key(c))
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
