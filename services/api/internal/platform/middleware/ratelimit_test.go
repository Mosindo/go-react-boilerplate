package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterLimitAndResets(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	current := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return current }

	for i := 0; i < 2; i++ {
		if ok, _ := limiter.Allow("a"); !ok {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	if ok, wait := limiter.Allow("a"); ok || wait <= 0 {
		t.Fatalf("third hit should be blocked with a wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := limiter.Allow("b"); !ok {
		t.Fatal("other keys must not be affected")
	}

	current = current.Add(61 * time.Second)
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("window should have reset")
	}
}
