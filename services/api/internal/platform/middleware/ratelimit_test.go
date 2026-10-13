package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindows(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rl := NewRateLimiter()
	rl.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := rl.Allow("k", 3, time.Minute); !ok {
			t.Fatalf("hit %d should pass", i)
		}
	}
	ok, retry := rl.Allow("k", 3, time.Minute)
	if ok || retry <= 0 || retry > time.Minute {
		t.Fatalf("4th hit must be blocked with a retry delay, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := rl.Allow("other", 3, time.Minute); !ok {
		t.Fatal("keys are independent")
	}
	now = now.Add(61 * time.Second)
	if ok, _ := rl.Allow("k", 3, time.Minute); !ok {
		t.Fatal("window must reset after the period")
	}
}
