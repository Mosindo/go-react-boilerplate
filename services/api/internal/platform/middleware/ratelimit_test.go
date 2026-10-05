package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := NewRateLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("ip"); !ok {
			t.Fatalf("hit %d should be allowed", i+1)
		}
	}
	ok, retry := l.Allow("ip")
	if ok || retry <= 0 || retry > time.Minute {
		t.Fatalf("4th hit must be refused with a retry delay, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("limits are per key")
	}

	now = now.Add(61 * time.Second)
	if ok, _ := l.Allow("ip"); !ok {
		t.Fatal("window must reset")
	}
}

func TestRateLimiterForgetsExpiredKeys(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := NewRateLimiter(1, time.Minute)
	l.now = func() time.Time { return now }
	for i := 0; i < 100; i++ {
		l.Allow(string(rune('a' + i)))
	}
	now = now.Add(5 * time.Minute)
	l.Allow("fresh")
	if len(l.buckets) != 1 {
		t.Fatalf("expired buckets must be collected, %d left", len(l.buckets))
	}
}
