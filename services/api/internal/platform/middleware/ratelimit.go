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

type limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
	lastGC  time.Time
	now     func() time.Time
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastGC) > time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.last) > time.Minute {
				delete(l.buckets, k)
			}
		}
		l.lastGC = now
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

// RateLimit is an in-memory token bucket (per API instance). ratePerSecond is the
// refill rate and burst the bucket size. keyFn selects the bucket; nil means client IP.
func RateLimit(ratePerSecond float64, burst int, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return rateLimitWithClock(ratePerSecond, burst, keyFn, time.Now)
}

func rateLimitWithClock(ratePerSecond float64, burst int, keyFn func(*gin.Context) string, now func() time.Time) gin.HandlerFunc {
	l := &limiter{buckets: map[string]*bucket{}, rate: ratePerSecond, burst: float64(burst), lastGC: now(), now: now}
	if keyFn == nil {
		keyFn = func(c *gin.Context) string { return c.ClientIP() }
	}
	return func(c *gin.Context) {
		if !l.allow(keyFn(c)) {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

// ByUser buckets requests per authenticated user (falls back to IP).
func ByUser(c *gin.Context) string {
	if id := c.GetString("userID"); id != "" {
		return "u:" + id
	}
	return "ip:" + c.ClientIP()
}
