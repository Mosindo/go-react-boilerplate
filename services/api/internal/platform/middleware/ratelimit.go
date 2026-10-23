package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter is an in-memory token bucket keyed by an arbitrary string.
// It is per-process: behind several API replicas the effective limit is multiplied.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	capacity float64
	now      func() time.Time
	lastGC   time.Time
}

// NewLimiter allows `perMinute` requests per key with a burst of the same size.
func NewLimiter(perMinute int) *Limiter {
	return &Limiter{
		buckets:  make(map[string]*bucket),
		rate:     float64(perMinute) / 60,
		capacity: float64(perMinute),
		now:      time.Now,
	}
}

// Allow consumes one token. When refused it returns the wait before the next token.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.gc(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.capacity, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// gc drops idle full buckets at most once a minute to bound memory.
func (l *Limiter) gc(now time.Time) {
	if now.Sub(l.lastGC) < time.Minute {
		return
	}
	l.lastGC = now
	for k, b := range l.buckets {
		if math.Min(l.capacity, b.tokens+now.Sub(b.last).Seconds()*l.rate) >= l.capacity {
			delete(l.buckets, k)
		}
	}
}

func reject(c *gin.Context, wait time.Duration) {
	secs := int(math.Ceil(wait.Seconds()))
	if secs < 1 {
		secs = 1
	}
	c.Header("Retry-After", strconv.Itoa(secs))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests", "code": "rate_limited"})
}

// RateLimitByIP throttles by client IP (use on unauthenticated endpoints).
// scope separates independent budgets (e.g. "auth").
func RateLimitByIP(l *Limiter, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, wait := l.Allow(scope + "|" + c.ClientIP()); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}

// RateLimitByUser throttles by authenticated user; place after RequireUser.
func RateLimitByUser(l *Limiter, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, wait := l.Allow(scope + "|" + c.GetString("userID")); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}
