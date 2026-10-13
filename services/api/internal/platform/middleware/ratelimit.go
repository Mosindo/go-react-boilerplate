package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a fixed-window in-memory limiter. It is per process: behind
// several API instances, enforce limits at the proxy as well.
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
	now     func() time.Time
}

type window struct {
	start time.Time
	count int
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{windows: make(map[string]*window), now: time.Now}
}

// Allow records one hit for key and reports whether it is within limit per period.
func (r *RateLimiter) Allow(key string, limit int, period time.Duration) (bool, time.Duration) {
	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.windows) > 50000 {
		for k, w := range r.windows {
			if now.Sub(w.start) > period {
				delete(r.windows, k)
			}
		}
	}

	w, ok := r.windows[key]
	if !ok || now.Sub(w.start) >= period {
		r.windows[key] = &window{start: now, count: 1}
		return true, 0
	}
	if w.count >= limit {
		return false, period - now.Sub(w.start)
	}
	w.count++
	return true, 0
}

// ByIP limits requests per client IP for a named bucket.
func (r *RateLimiter) ByIP(bucket string, limit int, period time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		r.enforce(c, bucket+":ip:"+c.ClientIP(), limit, period)
	}
}

// ByUser limits requests per authenticated user for a named bucket. It must run
// after RequireUser.
func (r *RateLimiter) ByUser(bucket string, limit int, period time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetString("userID")
		if key == "" {
			key = "ip:" + c.ClientIP()
		}
		r.enforce(c, bucket+":"+key, limit, period)
	}
}

func (r *RateLimiter) enforce(c *gin.Context, key string, limit int, period time.Duration) {
	ok, retryAfter := r.Allow(key, limit, period)
	if ok {
		c.Next()
		return
	}
	seconds := int(retryAfter.Seconds()) + 1
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
}
