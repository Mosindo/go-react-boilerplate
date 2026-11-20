package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory token bucket keyed by an arbitrary string
// (client IP or user id). It protects a single API instance; a shared store
// such as Redis is required to enforce limits across several instances.
type RateLimiter struct {
	mu        sync.Mutex
	rate      float64 // tokens per second
	burst     float64
	buckets   map[string]*bucket
	now       func() time.Time
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter allows `limit` events per `window`, with bursts up to `burst`.
func NewRateLimiter(limit int, window time.Duration, burst int) *RateLimiter {
	if burst <= 0 {
		burst = limit
	}
	return &RateLimiter{
		rate:    float64(limit) / window.Seconds(),
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
		now:     time.Now,
	}
}

// Allow consumes one token for key. When refused it returns the wait time.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = math.Min(l.burst, b.tokens+elapsed*l.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// sweep drops buckets that are full again so memory stays bounded.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	fullAfter := time.Duration(l.burst / l.rate * float64(time.Second))
	for key, b := range l.buckets {
		if now.Sub(b.last) > fullAfter {
			delete(l.buckets, key)
		}
	}
}

// KeyByIP rate limits by client IP (honouring only configured trusted proxies).
func KeyByIP(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

// KeyByUser rate limits by authenticated user, falling back to IP.
func KeyByUser(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "user:" + userID
	}
	return KeyByIP(c)
}

// RateLimit rejects requests above the limiter's budget with HTTP 429.
func RateLimit(limiter *RateLimiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, wait := limiter.Allow(key(c))
		if !allowed {
			seconds := int(math.Ceil(wait.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please slow down",
				"code":  "rate_limited",
			})
			return
		}
		c.Next()
	}
}
