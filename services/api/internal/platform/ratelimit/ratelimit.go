// Package ratelimit provides an in-memory token bucket limiter exposed as Gin
// middleware. It protects a single API instance against brute force and abuse.
// Running several instances behind a load balancer would require a shared
// store (e.g. Redis); the Limiter interface keeps that swap local.
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

// Limiter is a keyed token bucket: `burst` requests are allowed immediately
// and the bucket refills at `burst / per` tokens per second.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	burst   float64
	rate    float64
	now     func() time.Time
	lastGC  time.Time
}

func New(burst int, per time.Duration) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		burst:   float64(burst),
		rate:    float64(burst) / per.Seconds(),
		now:     time.Now,
	}
}

// Allow consumes a token for key. When refused it returns the duration after
// which a token becomes available.
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

// gc drops full buckets that have been idle long enough to refill.
func (l *Limiter) gc(now time.Time) {
	if now.Sub(l.lastGC) < time.Minute {
		return
	}
	l.lastGC = now
	refill := time.Duration(l.burst / l.rate * float64(time.Second))
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > refill {
			delete(l.buckets, key)
		}
	}
}

// KeyFunc derives the limiter key for a request.
type KeyFunc func(c *gin.Context) string

// ByIP keys on the client IP (honours only configured trusted proxies).
func ByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// ByUser keys on the authenticated user, falling back to IP.
func ByUser(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "user:" + userID
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
