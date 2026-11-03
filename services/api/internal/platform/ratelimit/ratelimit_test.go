package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterWindow(t *testing.T) {
	l := New(2, time.Minute)
	now := time.Now()
	l.now = func() time.Time { return now }
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("first two hits must pass")
	}
	if l.Allow("a") {
		t.Fatal("third hit must be limited")
	}
	if !l.Allow("b") {
		t.Fatal("other keys are independent")
	}
	now = now.Add(61 * time.Second)
	if !l.Allow("a") {
		t.Fatal("window must reset")
	}
}
