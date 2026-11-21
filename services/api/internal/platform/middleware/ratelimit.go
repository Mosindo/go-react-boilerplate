package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

// RateLimiter is a small in-memory fixed-window limiter. It protects abuse
// sensitive endpoints (auth, swipes, messages, uploads, reports) on a single
// API instance. A shared store (e.g. Redis) can replace it behind the same
// middleware signature when the API is scaled horizontally.
type RateLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	buckets   map[string]*rateBucket
	now       func() time.Time
	lastSweep time.Time
}

type rateBucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]*rateBucket),
		now:     time.Now,
	}
}

// Allow records a hit for key and reports whether it is within the limit and,
// if not, how long the caller should wait.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) > l.window {
		for k, b := range l.buckets {
			if now.After(b.resetAt) {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}

	bucket, ok := l.buckets[key]
	if !ok || now.After(bucket.resetAt) {
		bucket = &rateBucket{resetAt: now.Add(l.window)}
		l.buckets[key] = bucket
	}
	if bucket.count >= l.limit {
		return false, bucket.resetAt.Sub(now)
	}
	bucket.count++
	return true, 0
}

// KeyByIP keys requests by client IP (for unauthenticated endpoints). IPv6
// clients are bucketed by /64, the smallest block an end user usually gets,
// so rotating addresses within one allocation does not bypass limits.
func KeyByIP(c *gin.Context) string {
	return ipBucket(c.ClientIP())
}

func ipBucket(raw string) string {
	ip := net.ParseIP(raw)
	if ip == nil {
		return raw
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

// KeyByUser keys requests by authenticated user, falling back to IP.
func KeyByUser(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "u:" + userID
	}
	return "ip:" + c.ClientIP()
}

// RateLimit returns a middleware enforcing limiter for the key produced by keyFn.
// A nil limiter disables limiting (used by tests).
func RateLimit(limiter *RateLimiter, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil {
			c.Next()
			return
		}
		allowed, retryAfter := limiter.Allow(keyFn(c))
		if !allowed {
			seconds := int(retryAfter.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			httpx.AbortError(c, http.StatusTooManyRequests, "rate_limited", "too many requests, please slow down")
			return
		}
		c.Next()
	}
}
