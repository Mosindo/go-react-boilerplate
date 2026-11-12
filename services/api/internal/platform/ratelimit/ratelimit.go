// Package ratelimit provides a small in-memory sliding-window limiter used to
// protect sensitive endpoints (auth, swipes, messages, uploads, reports).
//
// It is process-local: when the API is scaled horizontally, replace the store
// with a shared backend (Redis, Postgres) behind the same Limiter interface.
package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Limiter interface {
	// Allow records a hit for key and reports whether it is within the quota,
	// plus how long the caller should wait before retrying when it is not.
	Allow(key string) (bool, time.Duration)
}

type window struct {
	start    time.Time
	previous int
	current  int
}

// Memory is a sliding-window counter (two fixed windows weighted together).
type Memory struct {
	mu        sync.Mutex
	limit     int
	period    time.Duration
	windows   map[string]*window
	now       func() time.Time
	lastSweep time.Time
}

func NewMemory(limit int, period time.Duration) *Memory {
	return &Memory{
		limit:   limit,
		period:  period,
		windows: make(map[string]*window),
		now:     time.Now,
	}
}

func (m *Memory) Allow(key string) (bool, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	m.sweep(now)

	w, ok := m.windows[key]
	if !ok {
		w = &window{start: now.Truncate(m.period)}
		m.windows[key] = w
	}

	currentStart := now.Truncate(m.period)
	switch elapsedWindows := int(currentStart.Sub(w.start) / m.period); {
	case elapsedWindows == 1:
		w.previous, w.current, w.start = w.current, 0, currentStart
	case elapsedWindows > 1:
		w.previous, w.current, w.start = 0, 0, currentStart
	}

	weight := 1 - float64(now.Sub(currentStart))/float64(m.period)
	estimated := float64(w.previous)*weight + float64(w.current)
	if estimated+1 > float64(m.limit) {
		return false, currentStart.Add(m.period).Sub(now)
	}
	w.current++
	return true, 0
}

func (m *Memory) sweep(now time.Time) {
	if now.Sub(m.lastSweep) < m.period {
		return
	}
	m.lastSweep = now
	for key, w := range m.windows {
		if now.Sub(w.start) > 2*m.period {
			delete(m.windows, key)
		}
	}
}

// KeyFunc derives the bucket key from a request.
type KeyFunc func(c *gin.Context) string

// ByIP buckets requests by client IP (honouring only trusted proxies).
func ByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// ByUser buckets requests by authenticated user, falling back to IP.
func ByUser(c *gin.Context) string {
	if id := c.GetString("userID"); id != "" {
		return "user:" + id
	}
	return ByIP(c)
}

// Middleware rejects requests over quota with 429 and a Retry-After header.
func Middleware(scope string, limiter Limiter, key KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, retryAfter := limiter.Allow(scope + "|" + key(c))
		if !ok {
			seconds := int(retryAfter.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please slow down"})
			return
		}
		c.Next()
	}
}
