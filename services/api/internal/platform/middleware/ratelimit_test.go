package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewRateLimiter(60, 3) // 1 token/second, burst 3
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d within burst must pass", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok || wait <= 0 {
		t.Fatalf("4th request must be limited with a wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys are independent")
	}
	now = now.Add(2 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("tokens refill over time")
	}
}
