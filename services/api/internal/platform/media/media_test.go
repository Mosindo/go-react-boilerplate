package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignerRoundTripAndTamper(t *testing.T) {
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"), time.Hour)
	key, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidKey(key) {
		t.Fatalf("generated key %q should be valid", key)
	}

	u, err := url.Parse(s.URL(key))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u.Path, "/media/") {
		t.Fatalf("unexpected path %q", u.Path)
	}
	exp, sig := u.Query().Get("exp"), u.Query().Get("sig")
	if !s.Verify(key, exp, sig) {
		t.Fatal("expected signature to verify")
	}
	if s.Verify(key, exp, strings.Repeat("0", len(sig))) {
		t.Fatal("tampered signature must fail")
	}
	other, _ := NewKey()
	if s.Verify(other, exp, sig) {
		t.Fatal("signature must be bound to key")
	}

	s.now = func() time.Time { return time.Now().Add(4 * time.Hour) }
	if s.Verify(key, exp, sig) {
		t.Fatal("expired link must fail")
	}
}

func TestValidKeyRejectsTraversal(t *testing.T) {
	for _, key := range []string{"../etc/passwd", "abc.jpg", "0123456789abcdef0123456789abcdef.png", "0123456789abcdef0123456789abcdef.jpg/../x"} {
		if ValidKey(key) {
			t.Fatalf("key %q must be rejected", key)
		}
	}
}

func TestLocalStorageSaveOpenDelete(t *testing.T) {
	store, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key, _ := NewKey()
	if err := store.Save(key, []byte("data")); err != nil {
		t.Fatal(err)
	}
	f, err := store.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := store.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(key); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := store.Save("../evil.jpg", []byte("x")); err != ErrInvalidKey {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
}
