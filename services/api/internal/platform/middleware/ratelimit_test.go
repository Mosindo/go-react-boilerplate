package middleware

import (
	"testing"
	"time"
)

func TestLimiterWindow(t *testing.T) {
	l := NewLimiter(2, time.Minute)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("hit %d should be allowed", i+1)
		}
	}
	if ok, retry := l.Allow("a"); ok || retry <= 0 {
		t.Fatalf("third hit must be rejected with a retry delay, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys must have their own budget")
	}

	now = now.Add(61 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("budget must reset after the window")
	}
}
