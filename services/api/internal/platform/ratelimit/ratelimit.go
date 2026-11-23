// Package ratelimit provides an in-memory token-bucket limiter used to protect
// abuse-prone endpoints (authentication, swipes, messages, uploads, reports).
//
// Limits are per API process. When running several replicas behind a load
// balancer, put a shared limiter (reverse proxy or Redis) in front as well.
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

// Limiter allows `burst` events immediately and refills at `perSecond`.
type Limiter struct {
	mu        sync.Mutex
	perSecond float64
	burst     float64
	buckets   map[string]*bucket
	now       func() time.Time
	lastSweep time.Time
}

// New builds a limiter allowing `events` per `window` with the same burst.
func New(events int, window time.Duration) *Limiter {
	return &Limiter{
		perSecond: float64(events) / window.Seconds(),
		burst:     float64(events),
		buckets:   make(map[string]*bucket),
		now:       time.Now,
	}
}

// Allow consumes a token for key. It returns false and the suggested retry
// delay when the bucket is empty.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastSeen).Seconds()
		b.tokens = math.Min(l.burst, b.tokens+elapsed*l.perSecond)
		b.lastSeen = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	missing := 1 - b.tokens
	return false, time.Duration(missing / l.perSecond * float64(time.Second))
}

// sweep drops idle full buckets to keep memory bounded.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	fullAfter := time.Duration(l.burst / l.perSecond * float64(time.Second))
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > fullAfter {
			delete(l.buckets, key)
		}
	}
}

// KeyFunc extracts the rate-limit key from a request.
type KeyFunc func(c *gin.Context) string

// ByIP keys requests by client IP.
func ByIP(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

// ByUser keys requests by authenticated user, falling back to IP.
func ByUser(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "user:" + userID
	}
	return ByIP(c)
}

// Middleware rejects requests over the limit with 429.
func Middleware(l *Limiter, scope string, key KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := l.Allow(scope + "|" + key(c))
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
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
