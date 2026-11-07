package storage

import "testing"

func TestDiskRoundTripAndKeyValidation(t *testing.T) {
	d, err := NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key, err := d.Put([]byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := d.Read(key); err != nil || string(got) != "data" {
		t.Fatalf("read back failed: %v", err)
	}
	for _, bad := range []string{"../etc/passwd", "a/b.jpg", "", "zz.jpg", key + "x"} {
		if _, err := d.Read(bad); err == nil {
			t.Errorf("key %q must be rejected", bad)
		}
	}
	d.Delete(key)
	if _, err := d.Read(key); err == nil {
		t.Fatal("deleted file must be gone")
	}
}
