package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalRoundTripAndTraversal(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := l.Put(ctx, "u1/a.jpg", strings.NewReader("data")); err != nil {
		t.Fatal(err)
	}
	rc, err := l.Open(ctx, "u1/a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "data" {
		t.Fatalf("got %q", b)
	}
	if err := l.Delete(ctx, "u1/a.jpg"); err != nil {
		t.Fatal(err)
	}
	if err := l.Delete(ctx, "u1/a.jpg"); err != nil {
		t.Fatalf("delete must be idempotent: %v", err)
	}
	if _, err := l.Open(ctx, "u1/a.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	for _, bad := range []string{"../etc/passwd", "/abs", "a/../../b", "a//b", ""} {
		if _, err := l.Put(ctx, bad, strings.NewReader("x")); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("key %q must be rejected, got %v", bad, err)
		}
	}
}
