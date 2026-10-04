package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory token bucket per key. It protects a single API instance;
// put a shared limiter (Redis, gateway) in front when scaling horizontally.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
	now     func() time.Time
	lastGC  time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter allows `burst` requests at once, refilled at `perMinute` requests per minute.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
		now:     time.Now,
	}
}

// Allow consumes one token for key. When it returns false, retryAfter says how long to wait.
func (l *RateLimiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.gc(now)

	b, found := l.buckets[key]
	if !found {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// gc drops idle buckets (full and untouched for a while) once per minute.
func (l *RateLimiter) gc(now time.Time) {
	if now.Sub(l.lastGC) < time.Minute {
		return
	}
	l.lastGC = now
	for key, b := range l.buckets {
		if now.Sub(b.last) > 10*time.Minute {
			delete(l.buckets, key)
		}
	}
}

// ByIP limits per client IP (gin's ClientIP honours the configured trusted proxies).
func (l *RateLimiter) ByIP() gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// ByUser limits per authenticated user (falls back to IP). Must run after RequireUser.
func (l *RateLimiter) ByUser() gin.HandlerFunc {
	return l.middleware(func(c *gin.Context) string {
		if id := c.GetString("userID"); id != "" {
			return "user:" + id
		}
		return "ip:" + c.ClientIP()
	})
}

func (l *RateLimiter) middleware(key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retry := l.Allow(key(c))
		if ok {
			c.Next()
			return
		}
		seconds := int(math.Ceil(retry.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		c.Header("Retry-After", strconv.Itoa(seconds))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests, slow down", "code": "rate_limited"})
	}
}
