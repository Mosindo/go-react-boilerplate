package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter is a small in-memory token bucket keyed by an arbitrary string.
// It is per-process: behind several API instances, enforce an additional limit at the proxy.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
	sweepAt time.Time
}

// NewLimiter allows `burst` requests at once and refills `perMinute` tokens each minute.
func NewLimiter(perMinute, burst int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
	}
}

// Allow consumes one token for key and reports whether the call is allowed.
func (l *Limiter) Allow(key string) bool {
	return l.allowAt(key, time.Now())
}

func (l *Limiter) allowAt(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.After(l.sweepAt) {
		for k, b := range l.buckets {
			if now.Sub(b.last) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.sweepAt = now.Add(time.Minute)
	}

	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.burst - 1, last: now}
		return true
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

func (l *Limiter) retryAfterSeconds() int {
	if l.rate <= 0 {
		return 60
	}
	secs := int(1/l.rate) + 1
	if secs < 1 {
		secs = 1
	}
	return secs
}

// ByIP limits requests per client IP (use on unauthenticated endpoints).
func (l *Limiter) ByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("ip:" + c.ClientIP()) {
			l.reject(c)
			return
		}
		c.Next()
	}
}

// ByUser limits requests per authenticated user; it must run after RequireUser.
func (l *Limiter) ByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("user:" + c.GetString("userID")) {
			l.reject(c)
			return
		}
		c.Next()
	}
}

func (l *Limiter) reject(c *gin.Context) {
	c.Header("Retry-After", strconv.Itoa(l.retryAfterSeconds()))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please slow down"})
}
