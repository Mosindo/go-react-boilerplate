package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a fixed-window in-memory limiter. It protects a single API instance against abuse
// (credential stuffing, spam); run a shared store (Redis) in front if you scale horizontally.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*bucket
	lastGC  time.Time
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, buckets: make(map[string]*bucket)}
}

// Allow records a hit for key and reports whether it is within the limit.
func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()

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
	b.count++
	if b.count > r.limit {
		return false, time.Until(b.resetAt)
	}
	return true, 0
}

// ByIP limits per client IP.
func (r *RateLimiter) ByIP() gin.HandlerFunc {
	return func(c *gin.Context) { r.check(c, "ip:"+c.ClientIP()) }
}

// ByUser limits per authenticated user; it must run after RequireUser.
func (r *RateLimiter) ByUser() gin.HandlerFunc {
	return func(c *gin.Context) { r.check(c, "u:"+c.GetString("userID")) }
}

func (r *RateLimiter) check(c *gin.Context, key string) {
	ok, retry := r.Allow(c.FullPath() + "|" + key)
	if ok {
		c.Next()
		return
	}
	c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
}
