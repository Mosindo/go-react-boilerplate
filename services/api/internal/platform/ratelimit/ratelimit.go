// Package ratelimit provides an in-memory fixed-window limiter suitable for a
// single API instance. Swap the Limiter for a shared store (Redis, Postgres)
// when running several replicas.
package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type window struct {
	start time.Time
	count int
}

// Limiter allows at most Max events per Window for each key.
type Limiter struct {
	Max    int
	Window time.Duration

	mu        sync.Mutex
	buckets   map[string]*window
	lastSweep time.Time
	now       func() time.Time
}

// New returns a limiter allowing max events per window and per key.
func New(max int, per time.Duration) *Limiter {
	return &Limiter{Max: max, Window: per, buckets: make(map[string]*window), now: time.Now}
}

// Allow records an event for key and reports whether it is within the quota,
// along with the time left before the window resets.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) > l.Window {
		for k, w := range l.buckets {
			if now.Sub(w.start) >= l.Window {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}

	w, ok := l.buckets[key]
	if !ok || now.Sub(w.start) >= l.Window {
		l.buckets[key] = &window{start: now, count: 1}
		return true, l.Window
	}
	remaining := l.Window - now.Sub(w.start)
	if w.count >= l.Max {
		return false, remaining
	}
	w.count++
	return true, remaining
}

// KeyFunc derives the limiting key from a request.
type KeyFunc func(c *gin.Context) string

// ByUserOrIP keys on the authenticated user when present, otherwise the client IP.
func ByUserOrIP(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "u:" + userID
	}
	return "ip:" + c.ClientIP()
}

// ByIP keys on the client IP only.
func ByIP(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

// Middleware rejects requests over quota with 429 and a Retry-After header.
func Middleware(l *Limiter, scope string, key KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := l.Allow(scope + "|" + key(c))
		if !allowed {
			seconds := int(retryAfter.Seconds()) + 1
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
