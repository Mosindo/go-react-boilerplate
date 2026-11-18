// Package ratelimit provides an in-memory token bucket limiter keyed by an
// arbitrary string (client IP, user id, ...). It is intentionally simple: a
// single API instance keeps its own buckets. Running several instances behind
// a load balancer multiplies the effective limit by the instance count, which
// remains acceptable for abuse protection; swap the Limiter for a shared store
// (Redis, Postgres) if strict global limits are ever required.
package ratelimit

import (
	"math"
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	ratePerS  float64
	burst     float64
	now       func() time.Time
	lastSweep time.Time
}

// New creates a limiter allowing `events` per `per` window with the given burst.
func New(events int, per time.Duration, burst int) *Limiter {
	if events <= 0 || per <= 0 {
		panic("ratelimit: events and per must be positive")
	}
	if burst <= 0 {
		burst = events
	}
	return &Limiter{
		buckets:  make(map[string]*bucket),
		ratePerS: float64(events) / per.Seconds(),
		burst:    float64(burst),
		now:      time.Now,
	}
}

// Allow consumes one token for key. When the bucket is empty it returns false
// and how long the caller should wait before retrying.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastSeen).Seconds()
		b.tokens = math.Min(l.burst, b.tokens+elapsed*l.ratePerS)
		b.lastSeen = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	missing := 1 - b.tokens
	wait := time.Duration(math.Ceil(missing/l.ratePerS*1000)) * time.Millisecond
	return false, wait
}

// sweep drops buckets that have fully refilled so memory stays bounded.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	fullAfter := time.Duration(l.burst/l.ratePerS*float64(time.Second)) + time.Second
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > fullAfter {
			delete(l.buckets, key)
		}
	}
}
