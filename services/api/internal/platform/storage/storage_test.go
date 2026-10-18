package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalPutOpenDelete(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("u1/a.jpg", []byte("data")); err != nil {
		t.Fatal(err)
	}
	f, err := s.Open("u1/a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(f)
	f.Close()
	if string(b) != "data" {
		t.Fatalf("got %q", b)
	}
	if err := s.Delete("u1/a.jpg", "u1/missing.jpg"); err != nil {
		t.Fatalf("deleting a missing file is not an error: %v", err)
	}
	if _, err := s.Open("u1/a.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestLocalRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	s, _ := NewLocal(filepath.Join(root, "uploads"))
	outside := filepath.Join(root, "secret.txt")
	_ = os.WriteFile(outside, []byte("secret"), 0o600)

	for _, key := range []string{"../secret.txt", "a/../../secret.txt", "/etc/passwd", "", ".."} {
		if err := s.Put(key, []byte("x")); err == nil {
			t.Errorf("Put(%q) must fail", key)
		}
		if _, err := s.Open(key); err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("Open(%q) must be rejected as invalid", key)
		}
	}
	if b, _ := os.ReadFile(outside); string(b) != "secret" {
		t.Fatal("file outside the root was modified")
	}
}
