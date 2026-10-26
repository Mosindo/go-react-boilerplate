package middleware

import (
	"testing"
	"time"
)

func TestLimiterBlocksAfterLimitAndRecovers(t *testing.T) {
	l := NewLimiter(2, 50*time.Millisecond)
	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	if ok, retry := l.Allow("k"); ok || retry <= 0 {
		t.Fatalf("third hit should be denied with retry delay, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := l.Allow("other"); !ok {
		t.Fatal("other key must not be affected")
	}
	time.Sleep(60 * time.Millisecond)
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("limit should reset after the window")
	}
}
