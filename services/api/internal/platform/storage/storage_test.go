package storage

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalStoreRoundTrip(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := store.Put(ctx, "photos/abc.jpg", []byte("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}
	f, err := store.Open(ctx, "photos/abc.jpg")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected content %q", data)
	}
	if err := store.Delete(ctx, "photos/abc.jpg"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Open(ctx, "photos/abc.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := store.Delete(ctx, "photos/abc.jpg"); err != nil {
		t.Fatalf("deleting a missing object must be a no-op: %v", err)
	}
}

func TestLocalStoreRejectsTraversal(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../etc/passwd.jpg", "a/../../b.jpg", "/abs/path.jpg", "", "noext", "photos/a b.jpg"} {
		if err := store.Put(context.Background(), key, []byte("x")); err == nil {
			t.Errorf("expected key %q to be rejected", key)
		}
	}
}
