package storage

import (
	"context"
	"os"
	"testing"
)

func TestLocalRoundTripAndKeyValidation(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := "0123456789abcdef0123456789abcdef.jpg"
	if err := s.Save(context.Background(), key, []byte("data")); err != nil {
		t.Fatalf("save: %v", err)
	}
	path, _ := s.Path(key)
	if b, err := os.ReadFile(path); err != nil || string(b) != "data" {
		t.Fatalf("read back failed: %v", err)
	}
	if err := s.Delete(context.Background(), key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	for _, bad := range []string{"../etc/passwd", "abc.jpg", "0123456789abcdef0123456789abcdef.php", ""} {
		if _, err := s.Path(bad); err == nil {
			t.Fatalf("expected key %q to be rejected", bad)
		}
	}
}
