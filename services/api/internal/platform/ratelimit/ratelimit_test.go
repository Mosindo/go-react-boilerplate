package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	l := New(60, 3)
	l.now = func() time.Time { return current }

	for i := 0; i < 3; i++ {
		if !l.Allow("ip") {
			t.Fatalf("expected event %d to be allowed", i)
		}
	}
	if l.Allow("ip") {
		t.Fatal("expected burst to be exhausted")
	}
	if !l.Allow("other") {
		t.Fatal("keys must be isolated")
	}

	current = current.Add(1100 * time.Millisecond)
	if !l.Allow("ip") {
		t.Fatal("expected one token after refill")
	}
	if l.Allow("ip") {
		t.Fatal("expected bucket empty again")
	}
}
