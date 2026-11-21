package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsWithinWindowAndResets(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	current := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return current }

	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("first hit should pass")
	}
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("second hit should pass")
	}
	ok, retry := limiter.Allow("a")
	if ok || retry <= 0 {
		t.Fatalf("third hit should be limited with retry delay, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := limiter.Allow("b"); !ok {
		t.Fatal("other keys are independent")
	}

	current = current.Add(61 * time.Second)
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("window should reset")
	}
}

func TestIPBucket(t *testing.T) {
	if ipBucket("203.0.113.7") != "203.0.113.7" {
		t.Fatal("IPv4 kept as-is")
	}
	if ipBucket("2001:db8:1:2:aaaa::1") != ipBucket("2001:db8:1:2:ffff::9") {
		t.Fatal("same /64 must share a bucket")
	}
	if ipBucket("2001:db8:1:2::1") == ipBucket("2001:db8:1:3::1") {
		t.Fatal("different /64 must differ")
	}
}
