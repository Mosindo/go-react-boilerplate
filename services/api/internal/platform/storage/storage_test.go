package storage

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalRoundTripAndTraversalProtection(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, "photos/ab/cdef.jpg", []byte("data")); err != nil {
		t.Fatalf("put: %v", err)
	}
	f, err := store.Open(ctx, "photos/ab/cdef.jpg")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	b, _ := io.ReadAll(f)
	_ = f.Close()
	if string(b) != "data" {
		t.Fatalf("unexpected content %q", b)
	}
	for _, bad := range []string{"../etc/passwd", "photos/../../x.jpg", "/abs.jpg", "Photos/X.jpg", ""} {
		if err := store.Put(ctx, bad, []byte("x")); err == nil {
			t.Fatalf("expected key %q to be rejected", bad)
		}
	}
	if err := store.Delete(ctx, "photos/ab/cdef.jpg"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Open(ctx, "photos/ab/cdef.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
