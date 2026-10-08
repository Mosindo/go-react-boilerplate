package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	count   int
	resetAt time.Time
}

// Limiter is a fixed-window in-memory rate limiter. It is per-process: run a shared
// store (Redis, ...) behind the same Allow signature if the API is scaled horizontally.
type Limiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*bucket
	lastGC  time.Time
	now     func() time.Time
}

func NewLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, buckets: map[string]*bucket{}, now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit,
// along with the time the caller should wait when it is not.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastGC) > l.window {
		for k, b := range l.buckets {
			if now.After(b.resetAt) {
				delete(l.buckets, k)
			}
		}
		l.lastGC = now
	}
	b, ok := l.buckets[key]
	if !ok || now.After(b.resetAt) {
		b = &bucket{resetAt: now.Add(l.window)}
		l.buckets[key] = b
	}
	if b.count >= l.limit {
		return false, b.resetAt.Sub(now)
	}
	b.count++
	return true, 0
}

// RateLimitByIP limits requests per client IP.
func RateLimitByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) { enforce(c, l, "ip:"+c.ClientIP()) }
}

// RateLimitByUser limits requests per authenticated user; it must run after RequireUser.
func RateLimitByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) { enforce(c, l, "user:"+c.GetString("userID")) }
}

func enforce(c *gin.Context, l *Limiter, key string) {
	ok, wait := l.Allow(key)
	if ok {
		c.Next()
		return
	}
	c.Header("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
}
