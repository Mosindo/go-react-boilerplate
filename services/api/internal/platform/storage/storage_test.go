package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalPutOpenDelete(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key, err := NewKey("photos/u1", ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, key, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	f, err := store.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected content %q", data)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestLocalRejectsTraversal(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../etc/passwd", "photos/../../x", ""} {
		if err := store.Put(context.Background(), key, []byte("x")); err == nil || !strings.Contains(err.Error(), "invalid storage key") {
			t.Fatalf("expected traversal rejection for %q, got %v", key, err)
		}
	}
}
