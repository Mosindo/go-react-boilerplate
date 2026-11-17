package storage

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalStoreRoundTripAndKeyValidation(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, "photos/abc-123.jpg", []byte("data")); err != nil {
		t.Fatalf("put: %v", err)
	}
	f, err := store.Open(ctx, "photos/abc-123.jpg")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	content, _ := io.ReadAll(f)
	f.Close()
	if string(content) != "data" {
		t.Fatalf("unexpected content %q", content)
	}
	if err := store.Delete(ctx, "photos/abc-123.jpg"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Open(ctx, "photos/abc-123.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	for _, bad := range []string{"../etc/passwd", "photos/../../x.jpg", "/abs.jpg", "photos/UPPER.jpg", "noext"} {
		if err := store.Put(ctx, bad, []byte("x")); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("expected invalid key for %q, got %v", bad, err)
		}
	}
}
