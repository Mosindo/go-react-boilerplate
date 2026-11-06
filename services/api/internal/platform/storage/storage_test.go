package storage

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalRoundTripAndKeyValidation(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key := "0123456789abcdef0123456789abcdef.jpg"
	if err := s.Put(ctx, key, []byte("data")); err != nil {
		t.Fatal(err)
	}
	f, err := s.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	_ = f.Close()
	if string(b) != "data" {
		t.Fatalf("got %q", b)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	for _, bad := range []string{"../etc/passwd", "a/b.jpg", "", "short.jpg", "0123456789abcdef0123456789abcdef.jpg/../x"} {
		if err := s.Put(ctx, bad, nil); err == nil {
			t.Errorf("key %q must be rejected", bad)
		}
	}
}
