// Package ratelimit provides an in-memory token-bucket limiter. It protects
// against abuse (credential stuffing, spam) and is deliberately generous so
// that normal use is never limited. State is per process.
package ratelimit

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

type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	burst    float64
	lastSwep time.Time
	now      func() time.Time
}

// New allows `burst` requests at once and refills `perMinute` tokens each minute.
func New(burst int, perMinute float64) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    perMinute / 60,
		burst:   float64(burst),
		now:     time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)

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

// sweep drops buckets that have fully refilled, bounding memory.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSwep) < time.Minute {
		return
	}
	l.lastSwep = now
	for k, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.burst {
			delete(l.buckets, k)
		}
	}
}

// ByIP limits per client IP (use for unauthenticated endpoints).
func ByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.ClientIP()) {
			tooMany(c)
			return
		}
		c.Next()
	}
}

// ByUser limits per authenticated user (mount after RequireUser).
func ByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.GetString("userID")) {
			tooMany(c)
			return
		}
		c.Next()
	}
}

func tooMany(c *gin.Context) {
	c.Header("Retry-After", "30")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
}
