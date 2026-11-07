package middleware

import (
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

// Limiter is a small in-memory token-bucket limiter keyed by an arbitrary string.
// It is per-process; put a shared limiter (e.g. at the proxy) in front for multi-instance deployments.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	burst    float64
	lastSwep time.Time
	now      func() time.Time
}

// NewLimiter allows `burst` requests at once, refilled evenly over `per`.
func NewLimiter(burst int, per time.Duration) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    float64(burst) / per.Seconds(),
		burst:   float64(burst),
		now:     time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastSwep) > time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.last) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.lastSwep = now
	}
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RateLimitByIP limits unauthenticated endpoints per client IP.
func RateLimitByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) { limit(c, l, "ip:"+c.ClientIP()) }
}

// RateLimitByUser limits authenticated endpoints per user (must run after RequireUser).
func RateLimitByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) { limit(c, l, "u:"+c.GetString("userID")) }
}

func limit(c *gin.Context, l *Limiter, key string) {
	if !l.Allow(key) {
		c.Header("Retry-After", strconv.Itoa(int(1/l.rate)+1))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
		return
	}
	c.Next()
}
