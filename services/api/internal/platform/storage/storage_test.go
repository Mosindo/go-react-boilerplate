package storage

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestLocalStoreRoundTripAndKeyValidation(t *testing.T) {
	s, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := "0123456789abcdef0123456789abcdef.jpg"
	if err := s.Put(context.Background(), key, []byte("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}
	f, err := s.Open(context.Background(), key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected content %q", data)
	}
	if err := s.Delete(context.Background(), key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Open(context.Background(), key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}

	for _, bad := range []string{"../../etc/passwd", "abc.jpg", "0123456789abcdef0123456789abcdef.php", ""} {
		if err := s.Put(context.Background(), bad, []byte("x")); err == nil {
			t.Fatalf("expected key %q to be rejected", bad)
		}
	}
}
