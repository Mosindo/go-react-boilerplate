package middleware

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, 50*time.Millisecond)
	for i := 0; i < 3; i++ {
		if ok, _ := rl.Allow("k"); !ok {
			t.Fatalf("hit %d should pass", i+1)
		}
	}
	if ok, retry := rl.Allow("k"); ok || retry <= 0 {
		t.Fatal("4th hit must be limited")
	}
	if ok, _ := rl.Allow("other"); !ok {
		t.Fatal("keys are independent")
	}
	time.Sleep(60 * time.Millisecond)
	if ok, _ := rl.Allow("k"); !ok {
		t.Fatal("window must reset")
	}
}
