package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBlocksAfterLimitAndResets(t *testing.T) {
	l := New(2, time.Minute)
	now := time.Unix(1_000, 0)
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	ok, wait := l.Allow("a")
	if ok || wait <= 0 || wait > time.Minute {
		t.Fatalf("third request should be blocked with a retry delay, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys must not be affected")
	}

	now = now.Add(time.Minute + time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("window should have reset")
	}
}
