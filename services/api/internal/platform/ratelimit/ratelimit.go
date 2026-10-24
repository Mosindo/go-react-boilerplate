// Package ratelimit provides an in-memory token-bucket limiter. It is per-process:
// run behind a shared limiter (gateway/Redis) if the API is scaled horizontally.
package ratelimit

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

type Limiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	buckets map[string]*bucket
	now     func() time.Time
	sweepAt time.Time
}

// New creates a limiter allowing `burst` requests at once, refilled at `perMinute` per minute.
func New(perMinute float64, burst int) *Limiter {
	return &Limiter{rate: perMinute / 60, burst: float64(burst), buckets: make(map[string]*bucket), now: time.Now}
}

// Allow consumes one token for key; it returns false and the retry delay when empty.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
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
		wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
		return false, wait
	}
	b.tokens--
	return true, 0
}

// sweep drops buckets idle long enough to be full again, bounding memory.
func (l *Limiter) sweep(now time.Time) {
	if now.Before(l.sweepAt) {
		return
	}
	l.sweepAt = now.Add(time.Minute)
	idle := time.Duration(l.burst/l.rate*float64(time.Second)) + time.Minute
	for key, b := range l.buckets {
		if now.Sub(b.last) > idle {
			delete(l.buckets, key)
		}
	}
}

// ByIP limits per client IP. ByUser limits per authenticated user (falls back to IP).
func (l *Limiter) ByIP() gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

func (l *Limiter) ByUser() gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string {
		if id := c.GetString("userID"); id != "" {
			return "user:" + id
		}
		return "ip:" + c.ClientIP()
	})
}

func (l *Limiter) middleware(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := l.Allow(c.FullPath() + "|" + key(c))
		if !ok {
			secs := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
