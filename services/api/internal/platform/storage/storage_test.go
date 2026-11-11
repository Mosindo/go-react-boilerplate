package storage

import (
	"errors"
	"io"
	"testing"
)

func TestLocalPutOpenDelete(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := "0123456789abcdef0123456789abcdef.jpg"
	if err := s.Put(key, []byte("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}
	f, err := s.Open(key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected content %q", data)
	}
	if err := s.Delete(key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Open(key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLocalRejectsTraversalKeys(t *testing.T) {
	s, _ := NewLocal(t.TempDir())
	for _, key := range []string{"../secret.jpg", "abc.jpg", "0123456789abcdef0123456789abcdef.php", "/etc/passwd"} {
		if err := s.Put(key, []byte("x")); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("key %q should be rejected, got %v", key, err)
		}
	}
}
