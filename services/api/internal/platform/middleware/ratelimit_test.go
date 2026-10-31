package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	l := NewRateLimiter(60, 3)
	now := time.Unix(1000, 0)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d should be allowed within burst", i)
		}
	}
	if l.Allow("k") {
		t.Fatal("4th request should be limited")
	}
	if !l.Allow("other") {
		t.Fatal("other keys must have their own bucket")
	}
	now = now.Add(2 * time.Second)
	if !l.Allow("k") {
		t.Fatal("tokens should refill over time")
	}
}
