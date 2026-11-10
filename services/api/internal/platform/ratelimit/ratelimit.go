// Package ratelimit provides an in-memory token-bucket limiter used to protect
// sensitive endpoints against abuse (credential stuffing, spam, scraping).
//
// It is intentionally process-local: for a multi-instance deployment put an
// edge rate limiter (reverse proxy / API gateway) in front, or swap this
// implementation for a shared store behind the same Middleware signature.
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

// Limiter allows `burst` events immediately and then refills at `perMinute`.
type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	burst     float64
	perSecond float64
	now       func() time.Time
	lastSweep time.Time
}

func New(perMinute, burst int) *Limiter {
	return &Limiter{
		buckets:   make(map[string]*bucket),
		burst:     float64(burst),
		perSecond: float64(perMinute) / 60,
		now:       time.Now,
	}
}

// Allow consumes one token for key. It returns false and the suggested wait
// when the bucket is empty.
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
	wait := time.Duration((1 - b.tokens) / l.perSecond * float64(time.Second))
	return false, wait
}

// sweep drops idle buckets so memory stays bounded by active clients.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	idle := time.Duration(l.burst/l.perSecond*float64(time.Second)) + time.Minute
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > idle {
			delete(l.buckets, key)
		}
	}
}

// KeyFunc extracts the identity a request is limited by.
type KeyFunc func(c *gin.Context) string

// ByIP limits by client IP (respecting the router's trusted proxy settings).
func ByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// ByUser limits by authenticated user and falls back to the client IP.
func ByUser(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "user:" + userID
	}
	return ByIP(c)
}

// Middleware rejects requests over the limit with 429 and a Retry-After header.
func Middleware(limiter *Limiter, scope string, key KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, wait := limiter.Allow(scope + "|" + key(c))
		if !allowed {
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
