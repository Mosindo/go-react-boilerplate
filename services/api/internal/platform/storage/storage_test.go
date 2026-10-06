package storage

import (
	"io"
	"testing"
)

func TestDiskRoundTripAndKeyValidation(t *testing.T) {
	d, err := NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := "ab/0a1b2c3d-0000-4000-8000-123456789abc.jpg"
	if err := d.Put(key, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	f, err := d.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(f)
	_ = f.Close()
	if string(got) != "hello" {
		t.Fatalf("unexpected content %q", got)
	}
	if err := d.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Open(key); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	for _, bad := range []string{"../etc/passwd", "ab/../../x.jpg", "/abs/path.jpg", ""} {
		if err := d.Put(bad, nil); err == nil {
			t.Fatalf("key %q must be rejected", bad)
		}
	}
}
