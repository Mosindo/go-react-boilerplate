package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter is an in-memory token bucket keyed by an arbitrary string.
// It protects a single API instance; put a shared limiter (gateway/Redis)
// in front when running several replicas.
type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	perMinute float64
	burst     float64
	lastSweep time.Time
	now       func() time.Time
}

func NewLimiter(perMinute, burst int) *Limiter {
	return &Limiter{
		buckets:   make(map[string]*bucket),
		perMinute: float64(perMinute),
		burst:     float64(burst),
		now:       time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Minutes() * l.perMinute
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

// sweep drops idle buckets so the map cannot grow without bound.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	for key, b := range l.buckets {
		if now.Sub(b.last) > 10*time.Minute {
			delete(l.buckets, key)
		}
	}
}

// RateLimitByIP limits unauthenticated endpoints per client IP.
func RateLimitByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("ip:" + c.ClientIP()) {
			abortTooMany(c)
			return
		}
		c.Next()
	}
}

// RateLimitByUser limits authenticated endpoints per user; must run after RequireUser.
func RateLimitByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("user:" + c.GetString("userID")) {
			abortTooMany(c)
			return
		}
		c.Next()
	}
}

func abortTooMany(c *gin.Context) {
	c.Header("Retry-After", "30")
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
}
