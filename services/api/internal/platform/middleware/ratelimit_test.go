package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsWithinWindowAndResets(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	current := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return current }

	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("first hit should pass")
	}
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("second hit should pass")
	}
	ok, retry := limiter.Allow("a")
	if ok || retry <= 0 {
		t.Fatalf("third hit should be limited with retry delay, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := limiter.Allow("b"); !ok {
		t.Fatal("other keys are independent")
	}

	current = current.Add(61 * time.Second)
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("window should reset")
	}
}
