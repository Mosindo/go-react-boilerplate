package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterWindow(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	l := New(2, time.Minute)
	l.now = func() time.Time { return current }

	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("first event should pass")
	}
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("second event should pass")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("third event should be limited")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys are independent")
	}

	current = current.Add(61 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("window should reset")
	}
}
