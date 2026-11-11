// Package ratelimit provides an in-memory token-bucket limiter keyed by an
// arbitrary string (client IP, user id...). It is process-local: when running
// several API replicas, put a shared limiter (reverse proxy, Redis) in front.
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

type Limiter struct {
	mu       sync.Mutex
	rate     float64 // tokens per second
	burst    float64
	buckets  map[string]*bucket
	now      func() time.Time
	lastGC   time.Time
	gcPeriod time.Duration
}

// New returns a limiter allowing `burst` requests at once, refilled at
// `perPeriod` requests per `period`.
func New(perPeriod int, period time.Duration, burst int) *Limiter {
	if burst < 1 {
		burst = 1
	}
	return &Limiter{
		rate:     float64(perPeriod) / period.Seconds(),
		burst:    float64(burst),
		buckets:  make(map[string]*bucket),
		now:      time.Now,
		gcPeriod: 5 * time.Minute,
	}
}

// Allow consumes one token for key and reports whether the call may proceed,
// along with the suggested wait before retrying when it may not.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.gc(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = minFloat(l.burst, b.tokens+elapsed*l.rate)
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	missing := 1 - b.tokens
	return false, time.Duration(missing / l.rate * float64(time.Second))
}

func (l *Limiter) gc(now time.Time) {
	if now.Sub(l.lastGC) < l.gcPeriod {
		return
	}
	l.lastGC = now
	idle := time.Duration(l.burst/l.rate*float64(time.Second)) + time.Minute
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > idle {
			delete(l.buckets, key)
		}
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
