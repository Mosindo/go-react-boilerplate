package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory fixed-window limiter. It protects endpoints against abuse
// (credential stuffing, spam); it is never used to restrict normal product usage.
// For a multi-instance deployment, replace it with a shared store (e.g. Redis) behind the same API.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*bucket
	now     func() time.Time
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, buckets: make(map[string]*bucket), now: time.Now}
}

// Allow records a hit for key and reports whether it is within the limit,
// along with the time until the window resets.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.buckets) > 10_000 {
		for k, b := range l.buckets {
			if now.After(b.resetAt) {
				delete(l.buckets, k)
			}
		}
	}

	b, ok := l.buckets[key]
	if !ok || now.After(b.resetAt) {
		b = &bucket{resetAt: now.Add(l.window)}
		l.buckets[key] = b
	}
	b.count++
	return b.count <= l.limit, b.resetAt.Sub(now)
}

// KeyByIP limits by client IP. KeyByUser limits by authenticated user and falls back to IP.
func KeyByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

func KeyByUser(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "user:" + userID
	}
	return KeyByIP(c)
}

func RateLimit(limiter *RateLimiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := limiter.Allow(c.FullPath() + "|" + key(c))
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Trop de requêtes. Réessayez dans quelques instants.",
				"code":  "rate_limited",
			})
			return
		}
		c.Next()
	}
}

// BodyLimit caps request bodies. Routes listed in overrides (keyed by Gin full path,
// e.g. "/photos") get their own limit; every other route gets defaultLimit.
func BodyLimit(defaultLimit int64, overrides map[string]int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := defaultLimit
		if v, ok := overrides[c.FullPath()]; ok {
			limit = v
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
