package middleware

import (
	"testing"
	"time"
)

func TestTokenBucketsRefillAndLimit(t *testing.T) {
	now := time.Unix(1000, 0)
	b := newTokenBuckets(60, 2)
	b.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := b.allow("k"); !ok {
			t.Fatalf("request %d should pass within burst", i)
		}
	}
	ok, wait := b.allow("k")
	if ok || wait <= 0 {
		t.Fatalf("third request should be limited with a wait, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := b.allow("other"); !ok {
		t.Fatal("other keys must have their own bucket")
	}

	now = now.Add(1100 * time.Millisecond)
	if ok, _ := b.allow("k"); !ok {
		t.Fatal("bucket should refill after waiting")
	}
}
