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

type limiter struct {
	mu        sync.Mutex
	hits      map[string]*window
	limit     int
	period    time.Duration
	lastSweep time.Time
}

// allow reports whether key may proceed, and how long to wait otherwise.
func (l *limiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) > l.period {
		for k, w := range l.hits {
			if now.Sub(w.start) >= l.period {
				delete(l.hits, k)
			}
		}
		l.lastSweep = now
	}
	w := l.hits[key]
	if w == nil || now.Sub(w.start) >= l.period {
		l.hits[key] = &window{start: now, count: 1}
		return true, 0
	}
	if w.count >= l.limit {
		return false, l.period - now.Sub(w.start)
	}
	w.count++
	return true, 0
}

// KeyFunc picks the bucket for a request.
type KeyFunc func(c *gin.Context) string

func ByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// ByUser falls back to the IP for unauthenticated requests.
func ByUser(c *gin.Context) string {
	if id := c.GetString("userID"); id != "" {
		return "user:" + id
	}
	return ByIP(c)
}

// RateLimit allows `limit` requests per `period` for each key (fixed window, in memory).
func RateLimit(limit int, period time.Duration, key KeyFunc) gin.HandlerFunc {
	l := &limiter{hits: make(map[string]*window), limit: limit, period: period}
	return func(c *gin.Context) {
		ok, retry := l.allow(key(c), time.Now())
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down", "code": "rate_limited"})
			return
		}
		c.Next()
	}
}
