package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory token bucket keyed by client IP or user id.
// It protects abuse-prone endpoints (auth, uploads, reports, messages) on a single
// instance; swap it for a shared store (e.g. Redis) when running several replicas.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity float64
	refill   float64 // tokens per second
	now      func() time.Time
	lastGC   time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter allows `burst` requests immediately, then `perWindow` requests per `window`.
func NewRateLimiter(burst int, perWindow int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*bucket),
		capacity: float64(burst),
		refill:   float64(perWindow) / window.Seconds(),
		now:      time.Now,
	}
}

// Allow consumes one token for key and reports whether the request may proceed,
// plus the delay before a token becomes available when it may not.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.gc(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = math.Min(l.capacity, b.tokens+elapsed*l.refill)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.refill * float64(time.Second))
	return false, wait
}

func (l *RateLimiter) gc(now time.Time) {
	if now.Sub(l.lastGC) < time.Minute {
		return
	}
	l.lastGC = now
	fullAfter := time.Duration(l.capacity / l.refill * float64(time.Second))
	for key, b := range l.buckets {
		if now.Sub(b.last) > fullAfter {
			delete(l.buckets, key)
		}
	}
}

// LimitByIP rate-limits anonymous endpoints by client IP.
func LimitByIP(l *RateLimiter, scope string) gin.HandlerFunc {
	return limit(l, func(c *gin.Context) string { return scope + ":ip:" + c.ClientIP() })
}

// LimitByUser rate-limits authenticated endpoints by user id (falls back to IP).
func LimitByUser(l *RateLimiter, scope string) gin.HandlerFunc {
	return limit(l, func(c *gin.Context) string {
		if userID := c.GetString("userID"); userID != "" {
			return scope + ":user:" + userID
		}
		return scope + ":ip:" + c.ClientIP()
	})
}

func limit(l *RateLimiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := l.Allow(key(c))
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Trop de tentatives. Réessaie dans quelques instants.",
				"code":  "rate_limited",
			})
			return
		}
		c.Next()
	}
}
