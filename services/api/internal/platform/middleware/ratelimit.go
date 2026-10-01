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

// Limiter is an in-memory token bucket keyed by an arbitrary string.
// It protects a single API instance; put a shared limiter in front of the API
// (gateway / Redis) when running several replicas.
type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	capacity  float64
	refill    float64 // tokens per second
	lastSweep time.Time
	now       func() time.Time
}

// NewLimiter allows `limit` events per `window`, with bursts up to `limit`.
func NewLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{
		buckets:  make(map[string]*bucket),
		capacity: float64(limit),
		refill:   float64(limit) / window.Seconds(),
		now:      time.Now,
	}
}

// Allow consumes one token for key; the second value is the suggested retry delay.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.refill
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.last = now

	if b.tokens < 1 {
		wait := time.Duration((1 - b.tokens) / l.refill * float64(time.Second))
		return false, wait
	}
	b.tokens--
	return true, 0
}

// sweep drops buckets that have fully refilled so the map cannot grow unbounded.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	fullAfter := time.Duration(l.capacity / l.refill * float64(time.Second))
	for key, b := range l.buckets {
		if now.Sub(b.last) > fullAfter {
			delete(l.buckets, key)
		}
	}
}

// RateLimitByIP limits requests per client IP. A nil limiter disables the check.
func RateLimitByIP(l *Limiter) gin.HandlerFunc {
	return rateLimit(l, func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// RateLimitByUser limits requests per authenticated user (falls back to IP).
func RateLimitByUser(l *Limiter) gin.HandlerFunc {
	return rateLimit(l, func(c *gin.Context) string {
		if id := c.GetString("userID"); id != "" {
			return "user:" + id
		}
		return "ip:" + c.ClientIP()
	})
}

func rateLimit(l *Limiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil {
			c.Next()
			return
		}
		ok, wait := l.Allow(key(c))
		if !ok {
			seconds := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
			return
		}
		c.Next()
	}
}
