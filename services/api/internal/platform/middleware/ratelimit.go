package middleware

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory token bucket keyed by client identity.
// It protects a single API instance; for multi-instance deployments put a
// shared limiter (reverse proxy or Redis) in front of the API as well.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	burst    float64
	lastSwep time.Time
	now      func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter allows `events` per `per` with bursts up to `burst`.
func NewRateLimiter(events int, per time.Duration, burst int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    float64(events) / per.Seconds(),
		burst:   float64(burst),
		now:     time.Now,
	}
}

// Allow consumes a token for key and returns whether the call may proceed and,
// when refused, how long the caller should wait.
func (l *RateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = math.Min(l.burst, b.tokens+elapsed*l.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// sweep drops idle buckets so memory stays bounded.
func (l *RateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSwep) < time.Minute {
		return
	}
	l.lastSwep = now
	idle := time.Duration(l.burst/l.rate*float64(time.Second)) + time.Minute
	for key, b := range l.buckets {
		if now.Sub(b.last) > idle {
			delete(l.buckets, key)
		}
	}
}

// RateLimitByIP limits by client IP (use for unauthenticated endpoints).
func RateLimitByIP(l *RateLimiter, scope string) gin.HandlerFunc {
	return rateLimit(l, func(c *gin.Context) string { return scope + "|ip|" + c.ClientIP() })
}

// RateLimitByUser limits by authenticated user, falling back to IP.
func RateLimitByUser(l *RateLimiter, scope string) gin.HandlerFunc {
	return rateLimit(l, func(c *gin.Context) string {
		if userID := c.GetString("userID"); userID != "" {
			return scope + "|user|" + userID
		}
		return scope + "|ip|" + c.ClientIP()
	})
}

func rateLimit(l *RateLimiter, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := l.Allow(keyFn(c))
		if !ok {
			seconds := int(math.Ceil(wait.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
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

// MaxBodyBytes caps request bodies to protect the server from oversized payloads.
func MaxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

// BodyLimits caps JSON bodies at jsonLimit and multipart uploads at
// multipartLimit (only photo routes accept multipart).
func BodyLimits(jsonLimit, multipartLimit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			limit := jsonLimit
			if strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
				limit = multipartLimit
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
