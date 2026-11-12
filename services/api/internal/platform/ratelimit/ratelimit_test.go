package ratelimit

import (
	"testing"
	"time"
)

func TestMemoryLimiterBlocksAndRecovers(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	m := NewMemory(3, time.Minute)
	m.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := m.Allow("k"); !ok {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	ok, retry := m.Allow("k")
	if ok || retry <= 0 {
		t.Fatalf("4th hit should be blocked with retry hint, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := m.Allow("other"); !ok {
		t.Fatal("other keys must not be affected")
	}

	now = now.Add(2*time.Minute + time.Second)
	if ok, _ := m.Allow("k"); !ok {
		t.Fatal("limiter should recover after the window passes")
	}
}
