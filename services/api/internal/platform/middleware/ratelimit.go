package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a fixed-window in-memory limiter. It protects a single API
// instance against abuse; behind several instances use a shared store instead.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*bucket
	now     func() time.Time
	sweepAt time.Time
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, buckets: make(map[string]*bucket), now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit and,
// if not, how long the caller should wait.
func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	if now.After(r.sweepAt) {
		for k, b := range r.buckets {
			if now.After(b.resetAt) {
				delete(r.buckets, k)
			}
		}
		r.sweepAt = now.Add(r.window)
	}
	b, ok := r.buckets[key]
	if !ok || now.After(b.resetAt) {
		b = &bucket{resetAt: now.Add(r.window)}
		r.buckets[key] = b
	}
	b.count++
	if b.count > r.limit {
		return false, b.resetAt.Sub(now)
	}
	return true, 0
}

// ByIP limits requests per client IP.
func (r *RateLimiter) ByIP() gin.HandlerFunc {
	return r.handler(func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// ByUser limits requests per authenticated user; it must run after RequireUser.
func (r *RateLimiter) ByUser() gin.HandlerFunc {
	return r.handler(func(c *gin.Context) string { return "user:" + c.GetString("userID") })
}

func (r *RateLimiter) handler(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := r.Allow(key(c))
		if !ok {
			secs := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
			return
		}
		c.Next()
	}
}
