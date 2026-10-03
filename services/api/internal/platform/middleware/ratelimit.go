package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

type limiter struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	perSecond  float64
	burst      float64
	lastPrune  time.Time
	pruneAfter time.Duration
}

func (l *limiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastPrune) > l.pruneAfter {
		for k, b := range l.buckets {
			if now.Sub(b.lastSeen) > l.pruneAfter {
				delete(l.buckets, k)
			}
		}
		l.lastPrune = now
	}

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.lastSeen).Seconds() * l.perSecond
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.lastSeen = now

	if b.tokens < 1 {
		wait := time.Duration((1 - b.tokens) / l.perSecond * float64(time.Second))
		return false, wait
	}
	b.tokens--
	return true, 0
}

// RateLimit is an in-memory token bucket limiter. burst requests are allowed at once
// and capacity refills at perSecond. The limiter is per process: behind several API
// instances, enforce a global limit at the proxy as well.
func RateLimit(perSecond float64, burst int, keyFn func(*gin.Context) string) gin.HandlerFunc {
	l := &limiter{
		buckets:    make(map[string]*bucket),
		perSecond:  perSecond,
		burst:      float64(burst),
		lastPrune:  time.Now(),
		pruneAfter: 10 * time.Minute,
	}
	return func(c *gin.Context) {
		key := keyFn(c)
		if key == "" {
			c.Next()
			return
		}
		ok, wait := l.allow(key, time.Now())
		if !ok {
			seconds := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
			return
		}
		c.Next()
	}
}

// ByIP keys the limiter on the client IP.
func ByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// ByUser keys the limiter on the authenticated user, falling back to the IP.
func ByUser(c *gin.Context) string {
	if id := c.GetString("userID"); id != "" {
		return "user:" + id
	}
	return ByIP(c)
}
