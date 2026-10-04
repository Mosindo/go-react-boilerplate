package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBurstRefillAndIsolation(t *testing.T) {
	l := NewRateLimiter(60, 3) // 1 token / second, burst 3
	now := time.Unix(1000, 0)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("burst request %d refused", i)
		}
	}
	ok, retry := l.Allow("a")
	if ok || retry <= 0 || retry > time.Second {
		t.Fatalf("4th request: ok=%v retry=%v", ok, retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys must not be affected")
	}
	now = now.Add(1100 * time.Millisecond)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("token not refilled")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("only one token should have refilled")
	}
}

func TestRateLimiterGarbageCollectsIdleKeys(t *testing.T) {
	l := NewRateLimiter(60, 1)
	now := time.Unix(1000, 0)
	l.now = func() time.Time { return now }
	l.Allow("old")
	now = now.Add(20 * time.Minute)
	l.Allow("new")
	if _, exists := l.buckets["old"]; exists {
		t.Fatal("idle bucket not collected")
	}
}
