package middleware

import (
	"testing"
	"time"
)

func TestLimiterWindow(t *testing.T) {
	l := NewLimiter(3, time.Minute)
	now := time.Unix(1000, 0)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("hit %d should pass", i)
		}
	}
	if ok, retry := l.Allow("a"); ok || retry < 1 || retry > 60 {
		t.Fatalf("4th hit should be limited with retry in (0,60], got ok=%v retry=%d", ok, retry)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys are independent")
	}
	now = now.Add(61 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("window should reset")
	}
}
