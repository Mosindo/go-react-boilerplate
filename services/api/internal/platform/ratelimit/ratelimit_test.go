package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	now := time.Unix(1000, 0)
	l := New(60, 3) // 1 token/second, burst 3
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("request %d should pass", i)
		}
	}
	if l.Allow("a") {
		t.Fatal("burst exhausted")
	}
	if !l.Allow("b") {
		t.Fatal("keys are independent")
	}
	now = now.Add(2 * time.Second)
	if !l.Allow("a") || !l.Allow("a") || l.Allow("a") {
		t.Fatal("expected exactly 2 refilled tokens")
	}
	now = now.Add(time.Hour)
	l.Allow("c") // triggers GC of idle buckets
	l.mu.Lock()
	n := len(l.buckets)
	l.mu.Unlock()
	if n != 1 {
		t.Fatalf("idle buckets should be collected, have %d", n)
	}
}
