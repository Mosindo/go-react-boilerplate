package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalStoreRoundTrip(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key, err := NewKey("jpg")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, key, strings.NewReader("data")); err != nil {
		t.Fatalf("put: %v", err)
	}
	rc, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	body, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(body) != "data" {
		t.Fatalf("unexpected body %q", body)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLocalStoreRejectsTraversal(t *testing.T) {
	store, _ := NewLocalStore(t.TempDir())
	for _, key := range []string{"../etc/passwd", "ab/../../x.jpg", "/abs.jpg", "ab/short.jpg"} {
		if err := store.Put(context.Background(), key, strings.NewReader("x")); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("expected ErrInvalidKey for %q, got %v", key, err)
		}
	}
}
