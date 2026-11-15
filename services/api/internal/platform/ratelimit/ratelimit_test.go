package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllowsBurstThenRefuses(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	ok, wait := l.Allow("k")
	if ok {
		t.Fatal("fourth request should be refused")
	}
	if wait <= 0 || wait > 21*time.Second {
		t.Fatalf("unexpected wait %v", wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("other keys must have their own bucket")
	}

	now = now.Add(20 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("bucket should have refilled one token")
	}
}
