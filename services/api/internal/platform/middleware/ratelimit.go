package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory token bucket keyed by an arbitrary string.
// It protects a single API instance against brute force and abuse; it is not
// a product quota (likes and messages are never artificially limited for
// normal use). Behind several instances, swap it for a shared store.
type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	rate      float64 // tokens per second
	burst     float64
	lastSweep time.Time
	now       func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter allows `burst` requests at once, refilled at `perWindow`
// requests every `window`.
func NewRateLimiter(perWindow int, window time.Duration, burst int) *RateLimiter {
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perWindow) / window.Seconds(),
		burst:   float64(burst),
		now:     time.Now,
	}
}

// Allow consumes a token for key and reports whether the request may proceed,
// plus the delay after which a retry would succeed.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = math.Min(l.burst, b.tokens+elapsed*l.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// sweep drops full buckets so memory stays proportional to active clients.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	for key, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.burst {
			delete(l.buckets, key)
		}
	}
}

// KeyFunc extracts the rate-limit key from a request.
type KeyFunc func(c *gin.Context) string

// ByIP keys on the client IP (respecting the configured trusted proxies only).
func ByIP(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

// ByUser keys on the authenticated user, falling back to the IP.
func ByUser(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "user:" + userID
	}
	return ByIP(c)
}

// RateLimit builds a Gin middleware from a limiter; scope namespaces keys so
// one limiter instance can never be confused with another.
func RateLimit(scope string, limiter *RateLimiter, key KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := limiter.Allow(scope + "|" + key(c))
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Trop de requêtes, réessaie dans quelques instants.",
				"code":  "rate_limited",
			})
			return
		}
		c.Next()
	}
}
