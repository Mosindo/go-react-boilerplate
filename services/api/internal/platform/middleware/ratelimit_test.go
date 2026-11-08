package middleware

import (
	"testing"
	"time"
)

func TestLimiterBlocksBurstAndRefills(t *testing.T) {
	l := NewLimiter(60, 3) // 1 token / second
	now := time.Now()
	for i := 0; i < 3; i++ {
		if !l.allowAt("k", now) {
			t.Fatalf("call %d should be allowed", i)
		}
	}
	if l.allowAt("k", now) {
		t.Fatal("4th immediate call should be blocked")
	}
	if !l.allowAt("k", now.Add(1100*time.Millisecond)) {
		t.Fatal("call after refill should be allowed")
	}
	if !l.allowAt("other", now) {
		t.Fatal("other keys are independent")
	}
}
