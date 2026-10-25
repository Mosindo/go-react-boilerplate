package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// MaxBody rejects request bodies larger than maxBytes.
func MaxBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large", "code": "payload_too_large"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// RateLimiter is a fixed-window in-memory limiter. It is per process: behind several replicas the
// effective limit is multiplied by the replica count.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*bucket
	lastGC  time.Time
	now     func() time.Time
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, buckets: make(map[string]*bucket), now: time.Now}
}

// Allow records one hit for key and reports whether it is within the limit, and the wait time otherwise.
func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	if now.Sub(r.lastGC) > r.window {
		for k, b := range r.buckets {
			if now.After(b.resetAt) {
				delete(r.buckets, k)
			}
		}
		r.lastGC = now
	}

	b := r.buckets[key]
	if b == nil || now.After(b.resetAt) {
		b = &bucket{resetAt: now.Add(r.window)}
		r.buckets[key] = b
	}
	if b.count >= r.limit {
		return false, b.resetAt.Sub(now)
	}
	b.count++
	return true, 0
}

// Middleware limits by client IP (before authentication) or by user id when one is set.
func (r *RateLimiter) Middleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := scope + ":" + c.ClientIP()
		if userID := c.GetString("userID"); userID != "" {
			key = scope + ":u:" + userID
		}
		ok, wait := r.Allow(key)
		if !ok {
			seconds := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
			return
		}
		c.Next()
	}
}
