package middleware

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	l := NewLimiter(60, 3) // 1 token per second, burst 3
	clock := time.Unix(1_700_000_000, 0)
	l.now = func() time.Time { return clock }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d should be allowed within burst", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok {
		t.Fatal("fourth request should be limited")
	}
	if wait <= 0 || wait > 2*time.Second {
		t.Fatalf("unexpected retry delay %s", wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("other keys must not be affected")
	}

	clock = clock.Add(1100 * time.Millisecond)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("token should have refilled after a second")
	}
}

func TestLimiterSweepsIdleBuckets(t *testing.T) {
	l := NewLimiter(60, 2)
	clock := time.Unix(1_700_000_000, 0)
	l.now = func() time.Time { return clock }
	l.Allow("a")
	clock = clock.Add(10 * time.Minute)
	l.Allow("b")
	if _, ok := l.buckets["a"]; ok {
		t.Fatal("idle bucket should have been swept")
	}
}
