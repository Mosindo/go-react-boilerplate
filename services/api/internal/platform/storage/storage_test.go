package storage

import (
	"io"
	"strings"
	"testing"
)

func TestLocalStoreRoundTripAndKeyValidation(t *testing.T) {
	s, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := strings.Repeat("ab", 16) + ".jpg"
	if err := s.Save(key, []byte("data")); err != nil {
		t.Fatal(err)
	}
	rc, err := s.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "data" {
		t.Fatalf("unexpected content %q", b)
	}
	if err := s.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(key); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := s.Delete(key); err != nil {
		t.Fatalf("deleting twice must be harmless: %v", err)
	}
	for _, bad := range []string{"../etc/passwd", "/etc/passwd", "abc.jpg", strings.Repeat("ab", 16) + ".php", strings.Repeat("AB", 16) + ".jpg", ""} {
		if err := s.Save(bad, []byte("x")); err == nil {
			t.Errorf("key %q must be rejected", bad)
		}
		if _, err := s.Open(bad); err == nil || err == ErrNotFound {
			t.Errorf("open %q must be rejected as invalid", bad)
		}
	}
}
