package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory token bucket keyed by an arbitrary string
// (client IP, user id...). It protects abuse-prone endpoints of a single API
// instance; put a shared limiter (gateway, Redis) in front for multi-instance deployments.
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

// NewRateLimiter allows bursts of `burst` requests and refills `burst` tokens every `per`.
func NewRateLimiter(burst int, per time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*bucket),
		capacity: float64(burst),
		refill:   float64(burst) / per.Seconds(),
		now:      time.Now,
	}
}

// Allow consumes one token for key and reports whether the request may proceed,
// along with the suggested retry delay when it may not.
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
	b.tokens = minFloat(l.capacity, b.tokens+elapsed*l.refill)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.refill * float64(time.Second))
	return false, wait
}

// gc drops idle buckets so memory stays bounded. Caller holds the lock.
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

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// KeyByIP limits per client IP.
func KeyByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// KeyByUser limits per authenticated user, falling back to IP.
func KeyByUser(c *gin.Context) string {
	if id := c.GetString("userID"); id != "" {
		return "user:" + id
	}
	return KeyByIP(c)
}

// RateLimit rejects requests with 429 once the limiter bucket for the derived key is empty.
func RateLimit(limiter *RateLimiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := limiter.Allow(key(c))
		if !ok {
			seconds := int(wait.Seconds()) + 1
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
