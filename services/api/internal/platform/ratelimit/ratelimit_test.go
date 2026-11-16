package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(1, time.Second, 3)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("expected burst request %d to pass", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 {
		t.Fatalf("expected 4th request to be limited, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be independent")
	}

	now = now.Add(time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("expected one token after refill")
	}
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("expected bucket to be empty again")
	}
}

func TestLimiterGarbageCollectsIdleBuckets(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(10, time.Second, 10)
	l.now = func() time.Time { return now }
	l.Allow("a")
	now = now.Add(10 * time.Minute)
	l.Allow("b")
	if _, exists := l.buckets["a"]; exists {
		t.Fatal("expected idle bucket to be collected")
	}
}
