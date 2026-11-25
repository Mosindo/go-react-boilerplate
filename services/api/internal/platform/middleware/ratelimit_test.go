package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	rl := &RateLimiter{limit: 2, window: time.Minute, buckets: map[string]*bucket{}}
	current := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rl.now = func() time.Time { return current }

	if ok, _ := rl.Allow("k"); !ok {
		t.Fatal("first hit should pass")
	}
	if ok, _ := rl.Allow("k"); !ok {
		t.Fatal("second hit should pass")
	}
	if ok, _ := rl.Allow("k"); ok {
		t.Fatal("third hit should be limited")
	}
	if ok, _ := rl.Allow("other"); !ok {
		t.Fatal("keys must be isolated")
	}
	current = current.Add(61 * time.Second)
	if ok, _ := rl.Allow("k"); !ok {
		t.Fatal("window should reset")
	}
}
