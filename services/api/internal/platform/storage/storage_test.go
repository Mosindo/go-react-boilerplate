package storage

import (
	"context"
	"io"
	"testing"
)

func TestLocalRoundTripAndKeyValidation(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s.Put(ctx, "ab/cdef.jpg", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	f, err := s.Open(ctx, "ab/cdef.jpg")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("got %q", data)
	}
	if err := s.Delete(ctx, "ab/cdef.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(ctx, "ab/cdef.jpg"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	for _, bad := range []string{"../etc/passwd", "/abs.jpg", "a/../../b.jpg", "x.JPG", "a//b.jpg"} {
		if err := s.Put(ctx, bad, nil); err == nil {
			t.Fatalf("key %q should be rejected", bad)
		}
	}
}
