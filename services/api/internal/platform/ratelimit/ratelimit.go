// Package ratelimit provides an in-memory token-bucket limiter middleware.
// It is per-process: behind several API replicas, add a shared limiter at the proxy.
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
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
	now     func() time.Time
	lastGC  time.Time
}

// New allows `burst` requests at once and refills `perMinute` tokens per minute.
func New(perMinute, burst int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
		now:     time.Now,
	}
}

// Allow consumes one token for key, reporting whether the request may proceed.
func (l *Limiter) Allow(key string) bool {
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

// ByIP limits per client IP.
func (l *Limiter) ByIP() gin.HandlerFunc {
	return func(c *gin.Context) { l.handle(c, "ip:"+c.ClientIP()) }
}

// ByUserOrIP limits per authenticated user when available, else per IP.
func (l *Limiter) ByUserOrIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if uid := c.GetString("userID"); uid != "" {
			l.handle(c, "u:"+uid)
			return
		}
		l.handle(c, "ip:"+c.ClientIP())
	}
}

func (l *Limiter) handle(c *gin.Context, key string) {
	if !l.Allow(key) {
		c.Header("Retry-After", "30")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		return
	}
	c.Next()
}
