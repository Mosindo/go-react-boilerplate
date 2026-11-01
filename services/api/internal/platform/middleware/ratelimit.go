package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	count   int
	resetAt time.Time
}

// RateLimiter est un limiteur a fenetre fixe en memoire (par instance).
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*bucket
	lastGC  time.Time
	now     func() time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, buckets: make(map[string]*bucket), now: time.Now}
}

// Allow consomme un jeton pour la cle et indique si la requete est autorisee.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastGC) > l.window {
		for k, b := range l.buckets {
			if now.After(b.resetAt) {
				delete(l.buckets, k)
			}
		}
		l.lastGC = now
	}
	b, ok := l.buckets[key]
	if !ok || now.After(b.resetAt) {
		l.buckets[key] = &bucket{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

// ByIP limite par adresse IP cliente.
func (l *RateLimiter) ByIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("ip:" + c.ClientIP()) {
			abortTooMany(c, l.window)
			return
		}
		c.Next()
	}
}

// ByUser limite par utilisateur authentifie (a placer apres RequireUser).
func (l *RateLimiter) ByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow("u:" + c.GetString("userID")) {
			abortTooMany(c, l.window)
			return
		}
		c.Next()
	}
}

func abortTooMany(c *gin.Context, retry time.Duration) {
	c.Header("Retry-After", itoa(int(retry.Seconds())))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
