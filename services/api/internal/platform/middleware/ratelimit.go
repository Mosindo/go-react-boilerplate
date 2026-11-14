package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a fixed-window, in-memory limiter. It protects endpoints
// against brute force and spam; it is not a product quota. For several API
// instances, replace it with a shared store (e.g. Redis) behind the same API.
type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string]*windowCounter
	now      func() time.Time
	lastGC   time.Time
}

type windowCounter struct {
	start time.Time
	count int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		window:   window,
		counters: make(map[string]*windowCounter),
		now:      time.Now,
	}
}

// Allow records a hit for key and reports whether it is within the limit,
// plus the time to wait before retrying when it is not.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastGC) > l.window {
		for k, counter := range l.counters {
			if now.Sub(counter.start) >= l.window {
				delete(l.counters, k)
			}
		}
		l.lastGC = now
	}

	counter, ok := l.counters[key]
	if !ok || now.Sub(counter.start) >= l.window {
		l.counters[key] = &windowCounter{start: now, count: 1}
		return true, 0
	}
	if counter.count >= l.limit {
		return false, counter.start.Add(l.window).Sub(now)
	}
	counter.count++
	return true, 0
}

// ByIP limits requests per client IP (use for unauthenticated endpoints).
func (l *RateLimiter) ByIP(scope string) gin.HandlerFunc {
	return l.handler(func(c *gin.Context) string { return scope + ":" + c.ClientIP() })
}

// ByUser limits requests per authenticated user (must run after RequireUser).
func (l *RateLimiter) ByUser(scope string) gin.HandlerFunc {
	return l.handler(func(c *gin.Context) string {
		if userID := c.GetString("userID"); userID != "" {
			return scope + ":u:" + userID
		}
		return scope + ":ip:" + c.ClientIP()
	})
}

func (l *RateLimiter) handler(keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := l.Allow(keyFn(c))
		if !allowed {
			seconds := int(retryAfter.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "trop de requêtes, réessayez dans quelques instants",
				"code":  "rate_limited",
			})
			return
		}
		c.Next()
	}
}
