// Package ratelimit provides a small in-memory fixed-window limiter exposed as Gin middleware.
// It is per-process: behind several API instances, enforce limits at the proxy as well.
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

type Limiter struct {
	limit  int
	period time.Duration
	mu     sync.Mutex
	keys   map[string]*window
	now    func() time.Time
	sweep  time.Time
}

func New(limit int, period time.Duration) *Limiter {
	return &Limiter{limit: limit, period: period, keys: make(map[string]*window), now: time.Now}
}

// Allow records one hit for key and reports whether it is within the limit,
// together with the time the caller should wait before retrying when it is not.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.sweep) > l.period {
		for k, w := range l.keys {
			if now.Sub(w.start) >= l.period {
				delete(l.keys, k)
			}
		}
		l.sweep = now
	}

	w, ok := l.keys[key]
	if !ok || now.Sub(w.start) >= l.period {
		l.keys[key] = &window{start: now, count: 1}
		return true, 0
	}
	if w.count >= l.limit {
		return false, w.start.Add(l.period).Sub(now)
	}
	w.count++
	return true, 0
}

// ByIP limits per client IP; use it on unauthenticated endpoints.
func ByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) { apply(c, l, "ip:"+c.ClientIP()) }
}

// ByUser limits per authenticated user (falls back to IP); it must run after RequireUser.
func ByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if uid := c.GetString("userID"); uid != "" {
			apply(c, l, "user:"+uid)
			return
		}
		apply(c, l, "ip:"+c.ClientIP())
	}
}

func apply(c *gin.Context, l *Limiter, key string) {
	ok, retry := l.Allow(key)
	if ok {
		c.Next()
		return
	}
	seconds := int(retry.Seconds()) + 1
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
}
