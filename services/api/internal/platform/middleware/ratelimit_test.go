package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	limiter := NewRateLimiter(60, time.Minute, 3)
	limiter.now = func() time.Time { return current }

	for i := 0; i < 3; i++ {
		if ok, _ := limiter.Allow("k"); !ok {
			t.Fatalf("request %d should be allowed within burst", i)
		}
	}
	ok, wait := limiter.Allow("k")
	if ok || wait <= 0 {
		t.Fatalf("4th request should be limited, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := limiter.Allow("other"); !ok {
		t.Fatal("keys must be isolated")
	}

	current = current.Add(1100 * time.Millisecond)
	if ok, _ := limiter.Allow("k"); !ok {
		t.Fatal("token should refill after one second at 60/min")
	}
}
