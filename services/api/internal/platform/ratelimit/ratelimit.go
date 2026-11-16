// Package ratelimit provides an in-memory token-bucket limiter and a Gin
// middleware. State is per process: behind several replicas, put a shared
// limiter (e.g. at the reverse proxy) in front as well.
package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// Limiter allows `burst` events immediately and then `rate` events per second.
type Limiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*bucket
	now     func() time.Time
	lastGC  time.Time
}

// New creates a limiter allowing `events` per `per` with a burst of `burst`.
func New(events int, per time.Duration, burst int) *Limiter {
	return &Limiter{
		rate:    float64(events) / per.Seconds(),
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
		now:     time.Now,
	}
}

// Allow consumes one token for key. When refused, it returns the duration to
// wait before the next token is available.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.gc(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastSeen).Seconds()
		b.tokens = math.Min(l.burst, b.tokens+elapsed*l.rate)
		b.lastSeen = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// gc drops buckets that have been idle long enough to be full again.
func (l *Limiter) gc(now time.Time) {
	if now.Sub(l.lastGC) < time.Minute {
		return
	}
	l.lastGC = now
	idle := time.Duration(l.burst/l.rate*float64(time.Second)) + time.Minute
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > idle {
			delete(l.buckets, key)
		}
	}
}

// KeyFunc derives the limiting key from a request.
type KeyFunc func(c *gin.Context) string

// ByIP keys on the client IP (see TRUSTED_PROXIES for proxy handling).
func ByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// ByUser keys on the authenticated user, falling back to the IP.
func ByUser(c *gin.Context) string {
	if id := c.GetString("userID"); id != "" {
		return "user:" + id
	}
	return ByIP(c)
}

// Middleware rejects requests over the limit with 429 and Retry-After.
func Middleware(l *Limiter, scope string, key KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := l.Allow(scope + "|" + key(c))
		if !ok {
			seconds := int(math.Ceil(wait.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
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
