package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalStoreRoundTripAndKeyValidation(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key := "0123456789abcdef0123456789abcdef.jpg"

	if err := store.Put(ctx, key, bytes.NewReader([]byte("data"))); err != nil {
		t.Fatalf("put: %v", err)
	}
	f, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	got, _ := io.ReadAll(f)
	_ = f.Close()
	if string(got) != "data" {
		t.Fatalf("unexpected content %q", got)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := store.Put(ctx, "../escape.jpg", bytes.NewReader(nil)); err == nil {
		t.Fatal("path traversal keys must be rejected")
	}
}
