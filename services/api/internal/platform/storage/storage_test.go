package storage

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalPutOpenDelete(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("new local: %v", err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, "photos/u1/p1.jpg", []byte("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}
	f, err := store.Open(ctx, "photos/u1/p1.jpg")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected data %q", data)
	}
	if err := store.Delete(ctx, "photos/u1/p1.jpg"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Open(ctx, "photos/u1/p1.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestLocalRejectsTraversal(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("new local: %v", err)
	}
	for _, key := range []string{"../etc/passwd", "/abs", "a/../../b", "a\\b", "", "a b"} {
		if err := store.Put(context.Background(), key, []byte("x")); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("key %q: expected ErrInvalidKey, got %v", key, err)
		}
	}
}
