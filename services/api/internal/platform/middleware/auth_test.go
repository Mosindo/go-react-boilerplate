package middleware

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestCachedValidator(t *testing.T) {
	var calls atomic.Int32
	active := true
	v := CachedValidator(func(context.Context, string) (bool, error) {
		calls.Add(1)
		return active, nil
	}, 50*time.Millisecond)

	for i := 0; i < 5; i++ {
		if ok, _ := v(context.Background(), "s1"); !ok {
			t.Fatal("session should be active")
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("answers must be cached, got %d lookups", calls.Load())
	}
	active = false
	time.Sleep(60 * time.Millisecond)
	if ok, _ := v(context.Background(), "s1"); ok {
		t.Fatal("revocation must be visible once the TTL elapsed")
	}

	uncached := CachedValidator(func(context.Context, string) (bool, error) { calls.Add(1); return true, nil }, 0)
	before := calls.Load()
	_, _ = uncached(context.Background(), "s")
	_, _ = uncached(context.Background(), "s")
	if calls.Load()-before != 2 {
		t.Fatal("zero TTL must disable caching")
	}
}
