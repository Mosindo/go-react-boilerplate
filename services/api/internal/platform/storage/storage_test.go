package storage

import (
	"context"
	"io"
	"testing"
)

func TestLocalStoreRoundTripAndPathTraversal(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key, err := NewKey("photos/u1", ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, key, []byte("data")); err != nil {
		t.Fatal(err)
	}
	f, err := store.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "data" {
		t.Fatalf("unexpected content %q", data)
	}
	for _, bad := range []string{"../etc/passwd", "/abs", `a\b`, ""} {
		if err := store.Put(ctx, bad, []byte("x")); err == nil {
			t.Fatalf("expected key %q to be rejected", bad)
		}
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(ctx, key); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
