package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// RateLimiter is an in-memory token bucket keyed by a caller-provided key.
// It is per-process; run behind a shared limiter if you scale horizontally.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	burst    float64
	lastScan time.Time
	now      func() time.Time
}

// NewRateLimiter allows `burst` requests at once, refilled at `perMinute`.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
		now:     time.Now,
	}
}

// Allow consumes one token for key and reports the wait before the next one.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastScan) > 5*time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.last) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.lastScan = now
	}
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
		return false, wait
	}
	b.tokens--
	return true, 0
}

func (l *RateLimiter) reject(c *gin.Context, wait time.Duration) {
	secs := int(wait.Seconds()) + 1
	c.Header("Retry-After", strconv.Itoa(secs))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
}

// ByIP limits per client IP (use on unauthenticated routes).
func (l *RateLimiter) ByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, wait := l.Allow(c.ClientIP()); !ok {
			l.reject(c, wait)
			return
		}
		c.Next()
	}
}

// ByUser limits per authenticated user; it must run after RequireUser.
func (l *RateLimiter) ByUser(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, wait := l.Allow(scope + ":" + c.GetString("userID")); !ok {
			l.reject(c, wait)
			return
		}
		c.Next()
	}
}
