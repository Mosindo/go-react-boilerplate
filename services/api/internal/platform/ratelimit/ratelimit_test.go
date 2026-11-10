package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := New(60, 3) // 1 token per second, burst of 3
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("expected request %d to be allowed", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok {
		t.Fatal("expected 4th request to be limited")
	}
	if wait <= 0 || wait > time.Second {
		t.Fatalf("unexpected wait %v", wait)
	}

	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be isolated")
	}

	now = now.Add(1100 * time.Millisecond)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("expected a token after refill")
	}
}

func TestLimiterSweepsIdleBuckets(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := New(60, 2)
	l.now = func() time.Time { return now }
	l.Allow("a")
	now = now.Add(10 * time.Minute)
	l.Allow("b")
	if _, exists := l.buckets["a"]; exists {
		t.Fatal("expected idle bucket to be swept")
	}
}
