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

// Limiter is an in-memory token bucket keyed by an arbitrary string. It
// protects a single API instance against abuse; run it behind a shared store
// if the API is scaled horizontally.
type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	perSecond float64
	burst     float64
	now       func() time.Time
	lastSweep time.Time
}

// NewLimiter allows perMinute sustained requests per key with the given burst.
func NewLimiter(perMinute, burst int) *Limiter {
	return &Limiter{
		buckets:   make(map[string]*bucket),
		perSecond: float64(perMinute) / 60.0,
		burst:     float64(burst),
		now:       time.Now,
	}
}

// Allow consumes one token for key. When the bucket is empty it reports how
// long to wait before retrying.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.perSecond)
	b.last = now

	if b.tokens < 1 {
		wait := time.Duration((1 - b.tokens) / l.perSecond * float64(time.Second))
		return false, wait
	}
	b.tokens--
	return true, 0
}

// sweep drops buckets that have been idle long enough to be full again.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	idle := time.Duration(l.burst/l.perSecond*float64(time.Second)) + time.Minute
	for key, b := range l.buckets {
		if now.Sub(b.last) > idle {
			delete(l.buckets, key)
		}
	}
}

func (l *Limiter) middleware(keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := l.Allow(keyFn(c))
		if ok {
			c.Next()
			return
		}
		seconds := int(math.Ceil(wait.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		c.Header("Retry-After", strconv.Itoa(seconds))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "too many requests, please slow down",
			"code":  "rate_limited",
		})
	}
}

// ByIP limits per client address; use it on unauthenticated endpoints.
func (l *Limiter) ByIP() gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// ByUser limits per authenticated user; mount it after RequireUser.
func (l *Limiter) ByUser() gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string { return "user:" + c.GetString("userID") })
}
