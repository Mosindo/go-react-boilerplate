// Package ratelimit provides a small in-memory fixed-window rate limiter.
//
// It is per-process: behind several API instances each instance enforces its
// own budget. Swap Limiter for a shared store (e.g. Redis) when scaling out.
package ratelimit

import (
	"sync"
	"time"
)

type window struct {
	start time.Time
	count int
}

type Limiter struct {
	mu      sync.Mutex
	limit   int
	period  time.Duration
	entries map[string]*window
	now     func() time.Time
	sweepAt time.Time
}

func New(limit int, period time.Duration) *Limiter {
	return &Limiter{
		limit:   limit,
		period:  period,
		entries: make(map[string]*window),
		now:     time.Now,
	}
}

// Allow records one hit for key and reports whether it is within budget along
// with the time to wait before retrying when it is not.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	w, ok := l.entries[key]
	if !ok || now.Sub(w.start) >= l.period {
		l.entries[key] = &window{start: now, count: 1}
		return true, 0
	}
	if w.count >= l.limit {
		return false, w.start.Add(l.period).Sub(now)
	}
	w.count++
	return true, 0
}

// sweep drops expired windows at most once per period so memory stays bounded.
func (l *Limiter) sweep(now time.Time) {
	if now.Before(l.sweepAt) {
		return
	}
	for key, w := range l.entries {
		if now.Sub(w.start) >= l.period {
			delete(l.entries, key)
		}
	}
	l.sweepAt = now.Add(l.period)
}
