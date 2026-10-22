package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	l := NewRateLimiter(3, time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("hit %d should pass", i)
		}
	}
	if ok, retry := l.Allow("k"); ok || retry <= 0 {
		t.Fatal("4th hit must be refused with a retry delay")
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("budgets are per key")
	}
	now = now.Add(61 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("budget must reset after the window")
	}
}
