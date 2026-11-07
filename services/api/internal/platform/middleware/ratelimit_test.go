package middleware

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	l := NewLimiter(3, time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d should pass", i)
		}
	}
	if l.Allow("k") {
		t.Fatal("4th request must be limited")
	}
	if !l.Allow("other") {
		t.Fatal("keys are independent")
	}
	now = now.Add(21 * time.Second) // 1 token refilled at 3/min
	if !l.Allow("k") {
		t.Fatal("token should have refilled")
	}
	if l.Allow("k") {
		t.Fatal("only one token refilled")
	}
}
