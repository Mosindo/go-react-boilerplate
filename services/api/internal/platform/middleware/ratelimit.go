package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var rateLimitingDisabled atomic.Bool

// SetRateLimiting switches every RateLimit middleware on or off. It exists so
// integration tests can create many accounts from one address; it is always
// on in the real server.
func SetRateLimiting(enabled bool) {
	rateLimitingDisabled.Store(!enabled)
}

type window struct {
	count   int
	resetAt time.Time
}

// RateLimiter is a fixed-window in-memory limiter. It protects a single API
// instance; run a shared limiter (reverse proxy or Redis) when scaling out.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	period  time.Duration
	windows map[string]window
	now     func() time.Time
	sweeps  int
}

func NewRateLimiter(limit int, period time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, period: period, windows: make(map[string]window), now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit,
// plus how long to wait when it is not.
func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	r.sweeps++
	if r.sweeps%1000 == 0 {
		for k, w := range r.windows {
			if now.After(w.resetAt) {
				delete(r.windows, k)
			}
		}
	}

	w, ok := r.windows[key]
	if !ok || now.After(w.resetAt) {
		w = window{resetAt: now.Add(r.period)}
	}
	if w.count >= r.limit {
		r.windows[key] = w
		return false, w.resetAt.Sub(now)
	}
	w.count++
	r.windows[key] = w
	return true, 0
}

// RateLimit limits requests per authenticated user, or per client IP when the
// request is anonymous. Each call builds an independent limiter.
func RateLimit(limit int, period time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(limit, period)
	return func(c *gin.Context) {
		if rateLimitingDisabled.Load() {
			c.Next()
			return
		}
		key := c.GetString("userID")
		if key == "" {
			key = "ip:" + c.ClientIP()
		}
		allowed, retryAfter := limiter.Allow(key)
		if allowed {
			c.Next()
			return
		}
		seconds := int(retryAfter.Seconds()) + 1
		c.Header("Retry-After", strconv.Itoa(seconds))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, retry later"})
	}
}
