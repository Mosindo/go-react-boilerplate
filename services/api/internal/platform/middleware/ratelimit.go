package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// RateLimiter is a per-key token bucket kept in memory. It protects a single
// API instance; put a shared limiter (reverse proxy / Redis) in front when
// scaling horizontally.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	burst    float64
	now      func() time.Time
	lastSwep time.Time
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

// Allow consumes one token for key and reports whether the call may proceed.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSwep) > 5*time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.last) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.lastSwep = now
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
		return false
	}
	b.tokens--
	return true
}

// ByIP limits requests per client IP.
func (l *RateLimiter) ByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("ip:" + c.ClientIP()) {
			tooMany(c)
			return
		}
		c.Next()
	}
}

// ByUser limits requests per authenticated user; it must run after RequireUser.
func (l *RateLimiter) ByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetString("userID")
		if key == "" {
			key = "ip:" + c.ClientIP()
		}
		if !l.Allow("user:" + key) {
			tooMany(c)
			return
		}
		c.Next()
	}
}

func tooMany(c *gin.Context) {
	c.Header("Retry-After", "30")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
}
