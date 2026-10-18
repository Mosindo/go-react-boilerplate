package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterMaxAndResets(t *testing.T) {
	l := NewRateLimiter(2, time.Minute, true)
	base := time.Now()
	l.now = func() time.Time { return base }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	if ok, retry := l.Allow("k"); ok || retry <= 0 {
		t.Fatalf("third hit should be blocked with retry, got ok=%v retry=%d", ok, retry)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("other key must not be affected")
	}
	l.now = func() time.Time { return base.Add(2 * time.Minute) }
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("window should have reset")
	}
}

func TestRateLimiterDisabledAllowsEverything(t *testing.T) {
	l := NewRateLimiter(1, time.Minute, false)
	for i := 0; i < 10; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatal("disabled limiter must allow")
		}
	}
}
