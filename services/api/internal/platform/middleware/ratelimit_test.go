package middleware

import (
	"testing"
	"time"
)

func TestLimiterWindow(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewLimiter(2, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("hit %d should pass", i)
		}
	}
	if ok, wait := l.Allow("a"); ok || wait <= 0 {
		t.Fatalf("third hit should be limited with a wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys are independent")
	}
	now = now.Add(61 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("window should reset")
	}
}
