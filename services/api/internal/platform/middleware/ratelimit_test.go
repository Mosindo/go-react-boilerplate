package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := NewRateLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d should pass", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 {
		t.Fatalf("4th request should be limited, wait=%v", wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be isolated")
	}

	now = now.Add(20 * time.Second) // refills one token (3 per minute)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("token should have been refilled")
	}
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("only one token should have been refilled")
	}
}

func TestRateLimiterGC(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := NewRateLimiter(2, time.Second)
	l.now = func() time.Time { return now }
	l.Allow("a")
	now = now.Add(2 * time.Minute)
	l.Allow("b")
	if _, exists := l.buckets["a"]; exists {
		t.Fatal("idle bucket should be collected")
	}
}
