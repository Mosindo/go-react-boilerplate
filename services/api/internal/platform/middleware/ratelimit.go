package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

// RateLimiter is an in-memory fixed-window limiter. It protects abuse-prone
// endpoints (auth, uploads, messages, reports) on a single API instance.
// Deployments with several instances should move this to a shared store.
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
	rl := &RateLimiter{limit: limit, window: window, buckets: make(map[string]*bucket), now: time.Now}
	go rl.janitor()
	return rl
}

// Allow records a hit for key and reports whether it is within the limit,
// along with the time the window resets.
func (rl *RateLimiter) Allow(key string) (bool, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.resetAt) {
		b = &bucket{count: 0, resetAt: now.Add(rl.window)}
		rl.buckets[key] = b
	}
	b.count++
	return b.count <= rl.limit, b.resetAt
}

func (rl *RateLimiter) janitor() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := rl.now()
		for key, b := range rl.buckets {
			if now.After(b.resetAt) {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// ByIP limits requests per client IP (honouring TRUSTED_PROXIES only).
func (rl *RateLimiter) ByIP(scope string) gin.HandlerFunc {
	return rl.handler(func(c *gin.Context) string { return scope + ":ip:" + c.ClientIP() })
}

// ByUser limits requests per authenticated user; it must run after RequireUser.
func (rl *RateLimiter) ByUser(scope string) gin.HandlerFunc {
	return rl.handler(func(c *gin.Context) string {
		if id := httpx.UserID(c); id != "" {
			return scope + ":user:" + id
		}
		return scope + ":ip:" + c.ClientIP()
	})
}

func (rl *RateLimiter) handler(keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, resetAt := rl.Allow(keyFn(c))
		if !ok {
			retry := int(time.Until(resetAt).Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, httpx.ErrorBody{Error: "too many requests, please try again later", Code: "rate_limited"})
			return
		}
		c.Next()
	}
}

// MaxJSONBody caps non-multipart request bodies. Multipart uploads apply
// their own, larger limit in the handler that accepts them.
func MaxJSONBody(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.ContentType() != "multipart/form-data" {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
