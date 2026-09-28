package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

// Limiter is an in-memory fixed-window rate limiter keyed by an arbitrary
// string (client IP, user id). It is per-process: with several API replicas
// each keeps its own counters. Stale windows are swept lazily.
type Limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	count       int
	windowStart time.Time
}

// NewLimiter allows `limit` events per `window` for each key.
func NewLimiter(limit int, window time.Duration) *Limiter {
	return newLimiter(limit, window, time.Now)
}

func newLimiter(limit int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{limit: limit, window: window, now: now, buckets: make(map[string]*bucket), lastSweep: now()}
}

// Allow records one event for key. When the key is over its limit it returns
// false and how long until the window resets.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) > l.window {
		for k, b := range l.buckets {
			if now.Sub(b.windowStart) >= l.window {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}

	b, ok := l.buckets[key]
	if !ok || now.Sub(b.windowStart) >= l.window {
		l.buckets[key] = &bucket{count: 1, windowStart: now}
		return true, 0
	}
	if b.count >= l.limit {
		return false, b.windowStart.Add(l.window).Sub(now)
	}
	b.count++
	return true, 0
}

func reject(c *gin.Context, retry time.Duration) {
	secs := int(math.Ceil(retry.Seconds()))
	if secs < 1 {
		secs = 1
	}
	c.Header("Retry-After", strconv.Itoa(secs))
	httpx.Abort(c, 429, httpx.CodeRateLimited, "too many requests, slow down")
}

// RateLimitByIP limits per client IP (see config.TrustedProxies for how the IP
// is derived).
func RateLimitByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, retry := l.Allow(c.ClientIP()); !ok {
			reject(c, retry)
			return
		}
		c.Next()
	}
}

// RateLimitByUser limits per authenticated user; it must run after RequireUser.
func RateLimitByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetString("userID")
		if key == "" {
			key = "ip:" + c.ClientIP()
		}
		if ok, retry := l.Allow(key); !ok {
			reject(c, retry)
			return
		}
		c.Next()
	}
}

// BodyLimit caps the request body with http.MaxBytesReader.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			httpx.Abort(c, 413, httpx.CodePayloadTooLarge, "request body too large")
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
