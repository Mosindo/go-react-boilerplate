package storage

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestLocalRoundTripAndTraversal(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := store.Put(ctx, "photos/u1/a.jpg", strings.NewReader("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}
	rc, err := store.Open(ctx, "photos/u1/a.jpg")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected content %q", data)
	}

	for _, bad := range []string{"../escape", "photos/../../etc/passwd", "", "/"} {
		if err := store.Put(ctx, bad, strings.NewReader("x")); err == nil {
			t.Fatalf("expected traversal key %q to be rejected", bad)
		}
	}

	if err := store.DeletePrefix(ctx, "photos/u1"); err != nil {
		t.Fatalf("delete prefix: %v", err)
	}
	if _, err := store.Open(ctx, "photos/u1/a.jpg"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
