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
// It is per-process: behind several instances, move it to a shared store.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	capacity float64
	lastGC   time.Time
	now      func() time.Time
}

// NewLimiter allows `burst` requests at once and `perMinute` sustained.
func NewLimiter(perMinute, burst int) *Limiter {
	return &Limiter{
		buckets:  make(map[string]*bucket),
		rate:     float64(perMinute) / 60.0,
		capacity: float64(burst),
		now:      time.Now,
	}
}

// Allow consumes one token for key and returns the seconds to wait when denied.
func (l *Limiter) Allow(key string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastGC) > 5*time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.last) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.lastGC = now
	}
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.last = now
	if b.tokens < 1 {
		wait := int((1-b.tokens)/l.rate) + 1
		return false, wait
	}
	b.tokens--
	return true, 0
}

// RateLimitByIP limits requests per client IP.
func RateLimitByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) { limit(c, l, "ip:"+c.ClientIP()) }
}

// RateLimitByUser limits requests per authenticated user (falls back to IP).
func RateLimitByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ip:" + c.ClientIP()
		if uid := c.GetString("userID"); uid != "" {
			key = "u:" + uid
		}
		limit(c, l, key)
	}
}

func limit(c *gin.Context, l *Limiter, key string) {
	ok, wait := l.Allow(key)
	if ok {
		c.Next()
		return
	}
	c.Header("Retry-After", strconv.Itoa(wait))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests", "code": "rate_limited"})
}
