package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	count   int
	resetAt time.Time
}

// RateLimiter is a fixed-window limiter kept in memory. It is per API instance:
// put a shared limiter (gateway/Redis) in front when running several replicas.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*bucket
	now     func() time.Time
	lastGC  time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, buckets: make(map[string]*bucket), now: time.Now}
}

// Allow records one hit for key and reports whether it is within the limit and how long to wait otherwise.
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

	b, ok := r.buckets[key]
	if !ok || now.After(b.resetAt) {
		b = &bucket{resetAt: now.Add(r.window)}
		r.buckets[key] = b
	}
	if b.count >= r.limit {
		return false, b.resetAt.Sub(now)
	}
	b.count++
	return true, 0
}

// ByIP limits requests per client IP. scope separates counters of different route groups.
func (r *RateLimiter) ByIP(scope string) gin.HandlerFunc {
	return func(c *gin.Context) { r.enforce(c, scope+"|"+c.ClientIP()) }
}

// ByUser limits requests per authenticated user (falls back to IP). It must run after RequireUser.
func (r *RateLimiter) ByUser(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetString("userID")
		if id == "" {
			id = c.ClientIP()
		}
		r.enforce(c, scope+"|"+id)
	}
}

func (r *RateLimiter) enforce(c *gin.Context, key string) {
	ok, retry := r.Allow(key)
	if ok {
		c.Next()
		return
	}
	seconds := int(retry.Seconds()) + 1
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests", "code": "rate_limited"})
}
