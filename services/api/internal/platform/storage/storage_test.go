package storage

import (
	"errors"
	"io"
	"testing"
)

func TestLocalStoragePutOpenDelete(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := "0f8fad5b-d9cb-469f-a165-70867728950e.jpg"
	if err := s.Put(key, []byte("data")); err != nil {
		t.Fatal(err)
	}
	f, err := s.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	f.Close()
	if string(b) != "data" {
		t.Fatalf("got %q", b)
	}
	if err := s.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := s.Delete(key); err != nil {
		t.Fatalf("deleting a missing key is a no-op: %v", err)
	}
}

func TestLocalStorageRejectsPathTraversal(t *testing.T) {
	s, _ := NewLocal(t.TempDir())
	for _, key := range []string{"../../etc/passwd", "a/b.jpg", "..", "", "0f8fad5b-d9cb-469f-a165-70867728950e.php", "/etc/passwd"} {
		if err := s.Put(key, nil); err == nil {
			t.Errorf("key %q must be rejected", key)
		}
		if _, err := s.Open(key); err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("open %q must be rejected as invalid", key)
		}
	}
}
