// Package ratelimit provides a small in-memory fixed-window limiter.
// It is per-process: behind several replicas use a shared store instead.
package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type window struct {
	start time.Time
	count int
}

type Limiter struct {
	mu      sync.Mutex
	limit   int
	period  time.Duration
	buckets map[string]*window
	lastGC  time.Time
	now     func() time.Time
}

func New(limit int, period time.Duration) *Limiter {
	return &Limiter{limit: limit, period: period, buckets: map[string]*window{}, now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastGC) > l.period {
		for k, w := range l.buckets {
			if now.Sub(w.start) >= l.period {
				delete(l.buckets, k)
			}
		}
		l.lastGC = now
	}
	w, ok := l.buckets[key]
	if !ok || now.Sub(w.start) >= l.period {
		l.buckets[key] = &window{start: now, count: 1}
		return true
	}
	if w.count >= l.limit {
		return false
	}
	w.count++
	return true
}

// ByIP limits requests per client IP.
func (l *Limiter) ByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("ip:" + c.ClientIP()) {
			tooMany(c)
			return
		}
		c.Next()
	}
}

// ByUser limits requests per authenticated user (use after RequireUser).
func (l *Limiter) ByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("user:" + c.GetString("userID")) {
			tooMany(c)
			return
		}
		c.Next()
	}
}

func tooMany(c *gin.Context) {
	c.Header("Retry-After", "60")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests", "code": "rate_limited"})
}
