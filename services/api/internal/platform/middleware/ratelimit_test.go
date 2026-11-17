package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	l := NewRateLimiter(2, 1, time.Second)
	l.now = func() time.Time { return current }

	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("first request should pass")
	}
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("second request should pass (burst)")
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 {
		t.Fatalf("third request should be limited, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be isolated")
	}

	current = current.Add(time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("request after refill should pass")
	}
}
