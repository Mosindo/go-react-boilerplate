package ratelimit

import (
	"testing"
	"time"
)

func TestBurstThenRefill(t *testing.T) {
	now := time.Unix(1000, 0)
	l := New(60, 3) // 1 token per second, burst 3
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d should pass within burst", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 {
		t.Fatalf("burst exhausted: expected rejection with delay, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys are independent")
	}
	now = now.Add(1100 * time.Millisecond)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("a token should have refilled after ~1s")
	}
}

func TestIdleBucketsAreSwept(t *testing.T) {
	now := time.Unix(1000, 0)
	l := New(60, 3)
	l.now = func() time.Time { return now }
	l.Allow("old")
	now = now.Add(time.Hour)
	l.Allow("new")
	if len(l.buckets) != 1 {
		t.Fatalf("expected idle bucket to be swept, have %d", len(l.buckets))
	}
}
