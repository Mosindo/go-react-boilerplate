package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	now := time.Now()
	rl.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := rl.Allow("k"); !ok {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	if ok, wait := rl.Allow("k"); ok || wait <= 0 {
		t.Fatalf("third hit should be limited, ok=%v wait=%v", ok, wait)
	}
	if ok, _ := rl.Allow("other"); !ok {
		t.Fatal("other key must not be limited")
	}

	now = now.Add(61 * time.Second)
	if ok, _ := rl.Allow("k"); !ok {
		t.Fatal("window should have reset")
	}
}
