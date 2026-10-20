package middleware

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	now := time.Now()
	l := NewLimiter(60, 3)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d within burst denied", i)
		}
	}
	if l.Allow("k") {
		t.Fatal("burst exceeded but allowed")
	}
	if !l.Allow("other") {
		t.Fatal("buckets must be independent")
	}
	now = now.Add(2 * time.Second) // 60/min = 1 per second
	if !l.Allow("k") || !l.Allow("k") {
		t.Fatal("tokens should have refilled")
	}
	if l.Allow("k") {
		t.Fatal("refill should be capped by elapsed time")
	}
}

func TestLimiterSweepsIdleBuckets(t *testing.T) {
	now := time.Now()
	l := NewLimiter(10, 1)
	l.now = func() time.Time { return now }
	l.Allow("old")
	now = now.Add(20 * time.Minute)
	l.Allow("new")
	if _, ok := l.buckets["old"]; ok {
		t.Fatal("idle bucket should be removed")
	}
}
