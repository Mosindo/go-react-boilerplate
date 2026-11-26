package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	current := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return current }

	for i := 0; i < 2; i++ {
		if ok, _ := limiter.Allow("k"); !ok {
			t.Fatalf("hit %d should be allowed", i+1)
		}
	}
	if ok, _ := limiter.Allow("k"); ok {
		t.Fatal("third hit should be rejected")
	}
	if ok, _ := limiter.Allow("other"); !ok {
		t.Fatal("keys must be isolated")
	}

	current = current.Add(61 * time.Second)
	if ok, _ := limiter.Allow("k"); !ok {
		t.Fatal("window should reset")
	}
}
