package blob

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStoreRoundTrip(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("pAbCdEf123456", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	f, err := store.Open("pAbCdEf123456")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected content %q", data)
	}
	if err := store.Delete("pAbCdEf123456"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("pAbCdEf123456"); err != nil {
		t.Fatalf("deleting a missing blob is not an error: %v", err)
	}
	if _, err := store.Open("pAbCdEf123456"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLocalStoreRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	store, _ := NewLocalStore(filepath.Join(root, "uploads"))
	for _, key := range []string{"../secret", "..", "a/b/c/d/e/f/g/h", "", "short", strings.Repeat("a", 200), "key with spaces", "..%2f..%2fetc"} {
		if err := store.Put(key, []byte("x")); err == nil {
			t.Errorf("key %q must be rejected", key)
		}
		if _, err := store.Open(key); err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("opening key %q must fail as invalid", key)
		}
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatalf("nothing may be written outside the upload dir: %v", entries)
	}
}
