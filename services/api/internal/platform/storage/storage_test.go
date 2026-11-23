package storage

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalRoundTripAndKeyValidation(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key := "photos/abc-123/9f2c.jpg"
	if err := store.Put(ctx, key, []byte("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}
	obj, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(obj)
	_ = obj.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected data %q", data)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	for _, bad := range []string{"../etc/passwd", "photos/../../x", "/abs/path", "photos", "Photos/A.jpg", "photos//x"} {
		if err := store.Put(ctx, bad, []byte("x")); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("expected ErrInvalidKey for %q, got %v", bad, err)
		}
	}
}
