package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// tokenBuckets is a small in-memory token-bucket limiter. It is per-process:
// behind several API instances each instance enforces its own budget, which is
// sufficient to blunt abuse (credential stuffing, scraping, spam).
type tokenBuckets struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*bucket
	calls   int
	now     func() time.Time
}

func newTokenBuckets(perMinute, burst int) *tokenBuckets {
	return &tokenBuckets{
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
		now:     time.Now,
	}
}

// allow consumes one token for key and, when refused, reports how long to wait.
func (t *tokenBuckets) allow(key string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	t.calls++
	if t.calls%1024 == 0 {
		t.sweep(now)
	}

	b, ok := t.buckets[key]
	if !ok {
		b = &bucket{tokens: t.burst, last: now}
		t.buckets[key] = b
	}
	b.tokens = math.Min(t.burst, b.tokens+now.Sub(b.last).Seconds()*t.rate)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / t.rate * float64(time.Second))
	return false, wait
}

func (t *tokenBuckets) sweep(now time.Time) {
	idle := time.Duration(t.burst/t.rate*float64(time.Second)) + time.Minute
	for key, b := range t.buckets {
		if now.Sub(b.last) > idle {
			delete(t.buckets, key)
		}
	}
}

// RateLimitByIP limits requests per client IP.
func RateLimitByIP(perMinute, burst int) gin.HandlerFunc {
	return rateLimit(newTokenBuckets(perMinute, burst), func(c *gin.Context) string { return c.ClientIP() })
}

// RateLimitByUser limits requests per authenticated user (falls back to IP).
// It must run after RequireUser.
func RateLimitByUser(perMinute, burst int) gin.HandlerFunc {
	return rateLimit(newTokenBuckets(perMinute, burst), func(c *gin.Context) string {
		if id := c.GetString("userID"); id != "" {
			return "u:" + id
		}
		return c.ClientIP()
	})
}

func rateLimit(buckets *tokenBuckets, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := buckets.allow(key(c))
		if ok {
			c.Next()
			return
		}
		retry := int(math.Ceil(wait.Seconds()))
		if retry < 1 {
			retry = 1
		}
		c.Header("Retry-After", strconv.Itoa(retry))
		c.Abort()
		platformerrors.Respond(c, http.StatusTooManyRequests, "rate_limited", "too many requests, slow down")
	}
}
