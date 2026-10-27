package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	now := time.Now()
	l := NewRateLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d should pass", i)
		}
	}
	if ok, retry := l.Allow("k"); ok || retry <= 0 {
		t.Fatalf("4th request must be limited, retry=%v", retry)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys are independent")
	}
	now = now.Add(61 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("window must reset")
	}
}
