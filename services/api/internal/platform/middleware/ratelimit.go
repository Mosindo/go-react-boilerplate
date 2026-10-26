package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Limiter is an in-memory sliding-window rate limiter. It is per process:
// behind several API replicas, put an edge limiter in front as well.
type Limiter struct {
	limit  int
	window time.Duration
	mu     sync.Mutex
	hits   map[string][]time.Time
	nextGC time.Time
}

func NewLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, hits: make(map[string][]time.Time)}
}

// Allow records a hit for key and reports whether it is within the limit.
// When denied it returns the duration after which a retry can succeed.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.After(l.nextGC) {
		for k, ts := range l.hits {
			if len(ts) == 0 || now.Sub(ts[len(ts)-1]) > l.window {
				delete(l.hits, k)
			}
		}
		l.nextGC = now.Add(l.window)
	}

	cutoff := now.Add(-l.window)
	ts := l.hits[key]
	i := 0
	for i < len(ts) && ts[i].Before(cutoff) {
		i++
	}
	ts = ts[i:]
	if len(ts) >= l.limit {
		l.hits[key] = ts
		return false, ts[0].Add(l.window).Sub(now)
	}
	l.hits[key] = append(ts, now)
	return true, 0
}

// RateLimitByIP limits requests per client IP.
func RateLimitByIP(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		reject(c, l, c.ClientIP())
	}
}

// RateLimitByUser limits requests per authenticated user; it must run after RequireUser.
func RateLimitByUser(l *Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetString("userID")
		if key == "" {
			key = c.ClientIP()
		}
		reject(c, l, key)
	}
}

func reject(c *gin.Context, l *Limiter, key string) {
	ok, retry := l.Allow(key)
	if ok {
		c.Next()
		return
	}
	seconds := int(retry.Seconds()) + 1
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
}
