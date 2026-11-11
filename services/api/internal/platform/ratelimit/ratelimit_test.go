package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	l := New(1, time.Second, 3)
	now := time.Unix(1_700_000_000, 0)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("ip"); !ok {
			t.Fatalf("request %d should be allowed within burst", i)
		}
	}
	ok, wait := l.Allow("ip")
	if ok || wait <= 0 {
		t.Fatalf("4th request should be limited, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("keys must be isolated")
	}

	now = now.Add(time.Second)
	if ok, _ := l.Allow("ip"); !ok {
		t.Fatal("token should refill after one period")
	}
}
