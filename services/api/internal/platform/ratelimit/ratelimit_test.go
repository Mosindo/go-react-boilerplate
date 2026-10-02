package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	l := New(3, 60)
	now := time.Unix(1000, 0)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("request %d should pass", i)
		}
	}
	if l.Allow("a") {
		t.Fatal("4th request should be limited")
	}
	if !l.Allow("b") {
		t.Fatal("other keys are independent")
	}
	now = now.Add(2 * time.Second)
	if !l.Allow("a") {
		t.Fatal("tokens should refill over time")
	}
}
