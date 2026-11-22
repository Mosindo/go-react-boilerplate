package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterRefillsOverTime(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewRateLimiter(2, time.Second, 2)
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("call %d should be allowed within burst", i)
		}
	}
	if ok, wait := l.Allow("k"); ok || wait <= 0 {
		t.Fatalf("third call should be limited with a positive wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatalf("keys must be isolated")
	}

	now = now.Add(600 * time.Millisecond)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatalf("bucket should have refilled one token")
	}
}
