// Package ratelimit provides an in-memory token-bucket limiter keyed by an
// arbitrary string (client IP, user id...). It protects abuse-prone endpoints
// without capping normal usage. For multi-instance deployments, swap it for a
// shared store (Redis, Postgres) behind the same Allow method.
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// Limiter allows `burst` events immediately, then refills at `perMinute`.
type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	rate      float64
	burst     float64
	idleTTL   time.Duration
	calls     int
	now       func() time.Time
	sweepEach int
}

func New(perMinute, burst int) *Limiter {
	if perMinute <= 0 {
		perMinute = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &Limiter{
		buckets:   make(map[string]*bucket),
		rate:      float64(perMinute) / 60.0,
		burst:     float64(burst),
		idleTTL:   15 * time.Minute,
		now:       time.Now,
		sweepEach: 1000,
	}
}

// Allow consumes one token for key and reports whether the event may proceed.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.calls++
	if l.calls%l.sweepEach == 0 {
		for k, b := range l.buckets {
			if now.Sub(b.lastSeen) > l.idleTTL {
				delete(l.buckets, k)
			}
		}
	}

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastSeen).Seconds()
		b.tokens += elapsed * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.lastSeen = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
