// Package ratelimit provides an in-memory token-bucket limiter used to protect
// abuse-prone endpoints (auth, messaging, uploads, reports).
//
// Limits here are anti-abuse safeguards, not product quotas: they are sized so
// a human using the app never reaches them. State is per API instance; put a
// shared limiter (e.g. at the gateway) in front when running several replicas.
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

type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	burst    float64
	now      func() time.Time
	lastGC   time.Time
	idleTTL  time.Duration
	disabled bool
}

// New creates a limiter allowing `events` per `per`, with bursts up to `burst`.
func New(events int, per time.Duration, burst int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    float64(events) / per.Seconds(),
		burst:   float64(burst),
		now:     time.Now,
		idleTTL: 2 * per,
	}
}

// Disabled returns a limiter that always allows requests (used by tests).
func Disabled() *Limiter {
	return &Limiter{disabled: true}
}

// Allow consumes one token for key and reports whether the event is allowed
// along with the suggested wait before retrying.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	if l.disabled {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.collectGarbage(now)

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

func (l *Limiter) collectGarbage(now time.Time) {
	if now.Sub(l.lastGC) < time.Minute {
		return
	}
	l.lastGC = now
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > l.idleTTL {
			delete(l.buckets, key)
		}
	}
}

// KeyFunc derives the bucket key for a request.
type KeyFunc func(c *gin.Context) string

// ByIP keys requests by client IP (see TRUSTED_PROXIES for proxy handling).
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

// Middleware rejects requests over the limit with 429 and a Retry-After header.
func Middleware(l *Limiter, scope string, key KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, wait := l.Allow(scope + "|" + key(c))
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
