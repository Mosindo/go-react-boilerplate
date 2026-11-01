package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	l := NewRateLimiter(3, time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d should pass", i)
		}
	}
	if l.Allow("k") {
		t.Fatal("4th request must be blocked")
	}
	if !l.Allow("other") {
		t.Fatal("other keys are independent")
	}
	now = now.Add(61 * time.Second)
	if !l.Allow("k") {
		t.Fatal("window should have reset")
	}
}
