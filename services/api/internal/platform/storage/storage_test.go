package storage

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestValidKey(t *testing.T) {
	for _, ok := range []string{"photos/ab12/3f2504e0-4f89-11d3-9a0c-0305e82c3301.jpg"} {
		if !ValidKey(ok) {
			t.Errorf("expected %q valid", ok)
		}
	}
	for _, bad := range []string{"../etc/passwd", "photos/../x.jpg", "/abs.jpg", "photos//x.jpg", "x", "photos/x.jpg/../../y.jpg"} {
		if ValidKey(bad) {
			t.Errorf("expected %q invalid", bad)
		}
	}
}

func TestLocalRoundTrip(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s.Put(ctx, "photos/aa/file.jpg", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	f, err := s.Open(ctx, "photos/aa/file.jpg")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	_ = f.Close()
	if string(b) != "hello" {
		t.Fatalf("got %q", b)
	}
	if err := s.Delete(ctx, "photos/aa/file.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(ctx, "photos/aa/file.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := s.Put(ctx, "../escape.jpg", []byte("x")); err == nil {
		t.Fatal("expected traversal key to be rejected")
	}
}
