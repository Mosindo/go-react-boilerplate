package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalRoundTripAndTraversal(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(filepath.Join(dir, "uploads"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s.Put(ctx, "photos/u1/a.jpg", []byte("data")); err != nil {
		t.Fatal(err)
	}
	f, err := s.Open(ctx, "photos/u1/a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	f.Close()
	if string(b) != "data" {
		t.Fatalf("got %q", b)
	}
	if err := s.Delete(ctx, "photos/u1/a.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(ctx, "photos/u1/a.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := s.Delete(ctx, "photos/u1/a.jpg"); err != nil {
		t.Fatalf("deleting a missing key is a no-op: %v", err)
	}

	secret := filepath.Join(dir, "secret.txt")
	_ = os.WriteFile(secret, []byte("x"), 0o600)
	for _, key := range []string{"../secret.txt", "photos/../../secret.txt", "/etc/passwd", "", "a\\b", "a\x00b", ".."} {
		if err := s.Put(ctx, key, []byte("pwn")); err == nil {
			t.Errorf("Put(%q) must be rejected", key)
		}
		if _, err := s.Open(ctx, key); err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("Open(%q) must be rejected as invalid, got %v", key, err)
		}
	}
	if b, _ := os.ReadFile(secret); string(b) != "x" {
		t.Fatal("file outside the root was modified")
	}
}
