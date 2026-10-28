package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type window struct {
	start time.Time
	count int
}

// Limiter is an in-memory fixed-window counter keyed by caller identity.
// It protects a single instance; put a shared limiter (e.g. at the proxy) in front when scaling out.
type Limiter struct {
	mu      sync.Mutex
	limit   int
	period  time.Duration
	windows map[string]*window
	lastGC  time.Time
	now     func() time.Time
}

func NewLimiter(limit int, period time.Duration) *Limiter {
	return &Limiter{limit: limit, period: period, windows: make(map[string]*window), now: time.Now}
}

// Allow records a hit and reports whether it is within the limit, plus seconds until the window resets.
func (l *Limiter) Allow(key string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastGC) > l.period {
		for k, w := range l.windows {
			if now.Sub(w.start) >= l.period {
				delete(l.windows, k)
			}
		}
		l.lastGC = now
	}

	w, ok := l.windows[key]
	if !ok || now.Sub(w.start) >= l.period {
		w = &window{start: now}
		l.windows[key] = w
	}
	w.count++
	if w.count > l.limit {
		retry := int(l.period.Seconds() - now.Sub(w.start).Seconds())
		if retry < 1 {
			retry = 1
		}
		return false, retry
	}
	return true, 0
}

// RateLimitByIP limits per client IP; RateLimitByUser per authenticated user (falls back to IP).
func RateLimitByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) { enforce(c, l, "ip:"+c.ClientIP()) }
}

func RateLimitByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ip:" + c.ClientIP()
		if uid := c.GetString("userID"); uid != "" {
			key = "user:" + uid
		}
		enforce(c, l, key)
	}
}

func enforce(c *gin.Context, l *Limiter, key string) {
	ok, retry := l.Allow(key)
	if ok {
		c.Next()
		return
	}
	c.Header("Retry-After", strconv.Itoa(retry))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
}
