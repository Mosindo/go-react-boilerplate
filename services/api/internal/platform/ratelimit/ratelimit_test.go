package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllowsBurstThenRefills(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	l := New(2, time.Second, 2)
	l.now = func() time.Time { return current }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("expected request %d to pass", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 {
		t.Fatalf("expected third request to be limited with a wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("expected independent key to pass")
	}

	current = current.Add(600 * time.Millisecond)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("expected refilled token to pass")
	}
}

func TestLimiterSweepsIdleBuckets(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	l := New(1, time.Second, 1)
	l.now = func() time.Time { return current }
	l.Allow("a")
	current = current.Add(2 * time.Minute)
	l.Allow("b")
	if _, ok := l.buckets["a"]; ok {
		t.Fatal("expected idle bucket to be swept")
	}
}
