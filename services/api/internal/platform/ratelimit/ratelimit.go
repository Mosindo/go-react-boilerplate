// Package ratelimit provides a small in-memory fixed-window rate limiter and
// Gin middlewares built on it. State is per process: behind several API
// instances, put a shared limiter (or the load balancer) in front instead.
package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type window struct {
	count   int
	resetAt time.Time
}

// Limiter allows at most Limit events per key within Window.
type Limiter struct {
	Limit  int
	Window time.Duration

	mu      sync.Mutex
	entries map[string]window
	now     func() time.Time
}

func New(limit int, per time.Duration) *Limiter {
	return &Limiter{Limit: limit, Window: per, entries: make(map[string]window), now: time.Now}
}

// Allow records an event for key and reports whether it is within the limit.
// When it is not, retryAfter says how long the caller must wait.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) > 10_000 {
		for k, w := range l.entries {
			if !now.Before(w.resetAt) {
				delete(l.entries, k)
			}
		}
	}

	w, ok := l.entries[key]
	if !ok || !now.Before(w.resetAt) {
		w = window{resetAt: now.Add(l.Window)}
	}
	if w.count >= l.Limit {
		l.entries[key] = w
		return false, w.resetAt.Sub(now)
	}
	w.count++
	l.entries[key] = w
	return true, 0
}

func reject(c *gin.Context, retryAfter time.Duration) {
	seconds := int(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	c.Header("Retry-After", strconv.Itoa(seconds))
	httpx.Fail(c, http.StatusTooManyRequests, httpx.CodeRateLimited, "too many requests, slow down")
}

// ByIP limits requests per client IP. name namespaces the counter.
func ByIP(name string, l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, wait := l.Allow(name + ":" + c.ClientIP()); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}

// ByUser limits requests per authenticated user. It must run after RequireUser.
func ByUser(name string, l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetString("userID")
		if key == "" {
			key = c.ClientIP()
		}
		if ok, wait := l.Allow(name + ":" + key); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}
