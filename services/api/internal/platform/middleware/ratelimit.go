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

// RateLimiter is a fixed-window in-memory limiter. It is per process: behind several
// replicas, enforce a stricter limit at the proxy or swap the store for Redis.
type RateLimiter struct {
	mu      sync.Mutex
	max     int
	period  time.Duration
	windows map[string]*window
}

func NewRateLimiter(max int, period time.Duration) *RateLimiter {
	return &RateLimiter{max: max, period: period, windows: make(map[string]*window)}
}

// Allow records a hit for key and reports whether it is within budget, plus the
// time to wait when it is not.
func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.windows) > 50000 {
		for k, w := range r.windows {
			if now.After(w.resetAt) {
				delete(r.windows, k)
			}
		}
	}

	w, ok := r.windows[key]
	if !ok || now.After(w.resetAt) {
		r.windows[key] = &window{count: 1, resetAt: now.Add(r.period)}
		return true, 0
	}
	if w.count >= r.max {
		return false, time.Until(w.resetAt)
	}
	w.count++
	return true, 0
}

// ByIP keys requests by client IP (configure Gin trusted proxies so it cannot be spoofed).
func ByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// ByUser keys requests by authenticated user, falling back to IP.
func ByUser(c *gin.Context) string {
	if id := c.GetString("userID"); id != "" {
		return "user:" + id
	}
	return ByIP(c)
}

// Limit builds a middleware enforcing max requests per period for the key function.
func Limit(max int, period time.Duration, key func(*gin.Context) string) gin.HandlerFunc {
	limiter := NewRateLimiter(max, period)
	return func(c *gin.Context) {
		ok, wait := limiter.Allow(key(c))
		if !ok {
			seconds := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
			return
		}
		c.Next()
	}
}
