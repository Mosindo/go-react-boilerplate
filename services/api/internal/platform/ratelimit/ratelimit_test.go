package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBlocksAfterLimitAndResetsAfterWindow(t *testing.T) {
	l := New(2, time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	if ok, retry := l.Allow("k"); ok || retry <= 0 {
		t.Fatalf("third hit must be blocked with retry delay, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("other keys must be independent")
	}

	now = now.Add(time.Minute + time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("window should have reset")
	}
}
