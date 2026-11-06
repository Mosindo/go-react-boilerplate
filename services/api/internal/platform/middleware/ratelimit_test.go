package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	now := time.Unix(1000, 0)
	rl.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := rl.Allow("a"); !ok {
			t.Fatalf("hit %d should pass", i)
		}
	}
	if ok, wait := rl.Allow("a"); ok || wait <= 0 {
		t.Fatalf("third hit must be blocked, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := rl.Allow("b"); !ok {
		t.Fatal("other keys are independent")
	}
	now = now.Add(2 * time.Minute)
	if ok, _ := rl.Allow("a"); !ok {
		t.Fatal("window should reset")
	}
}
