package storage

import (
	"errors"
	"io"
	"testing"
)

func TestDiskRoundTripAndTraversalRejection(t *testing.T) {
	disk, err := NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := NewKey([]byte("0123456789abcdef0123456789abcdef"))

	if err := disk.Put(key, []byte("hello")); err != nil {
		t.Fatalf("put: %v", err)
	}
	f, err := disk.Open(key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected data %q", data)
	}

	if err := disk.Delete(key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := disk.Open(key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	for _, bad := range []string{"../etc/passwd", "ab/../../x.jpg", "/abs.jpg", ""} {
		if err := disk.Put(bad, []byte("x")); err == nil {
			t.Fatalf("key %q should be rejected", bad)
		}
	}
}
