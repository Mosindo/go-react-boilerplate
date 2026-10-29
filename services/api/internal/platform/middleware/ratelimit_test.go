package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterLimitAndResets(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewRateLimiter(2, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	if ok, retry := l.Allow("a"); ok || retry <= 0 {
		t.Fatalf("third hit should be blocked with retry-after, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys must not be affected")
	}

	now = now.Add(61 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("window should have reset")
	}
}
