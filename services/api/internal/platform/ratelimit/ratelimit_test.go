package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllowsBurstThenRefills(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := New(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("expected request %d to be allowed", i)
		}
	}
	ok, retry := l.Allow("k")
	if ok {
		t.Fatal("expected 4th request to be limited")
	}
	if retry <= 0 || retry > 20*time.Second+time.Millisecond {
		t.Fatalf("unexpected retry duration %s", retry)
	}

	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("expected independent key to be allowed")
	}

	now = now.Add(20 * time.Second)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("expected a token to be refilled after 20s")
	}
}
