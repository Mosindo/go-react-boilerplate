package storage

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestSignerRoundTrip(t *testing.T) {
	s := NewSigner([]byte("signer-secret-0123456789-abcdefghijkl"))
	u := s.URL("photo-1", "thumb")
	if !strings.HasPrefix(u, "/media/photos/photo-1?") {
		t.Fatalf("unexpected url %s", u)
	}
	q := map[string]string{}
	for _, kv := range strings.Split(strings.SplitN(u, "?", 2)[1], "&") {
		p := strings.SplitN(kv, "=", 2)
		q[p[0]] = p[1]
	}
	if !s.Verify("photo-1", "thumb", q["exp"], q["sig"]) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("photo-2", "thumb", q["exp"], q["sig"]) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", "full", q["exp"], q["sig"]) {
		t.Fatal("signature must be bound to the size")
	}
	if s.Verify("photo-1", "thumb", "1", q["sig"]) {
		t.Fatal("signature must be bound to the expiry")
	}
	expired := s.sign("photo-1", "thumb", time.Now().Add(-time.Minute).Unix())
	if s.Verify("photo-1", "thumb", itoa(time.Now().Add(-time.Minute).Unix()), expired) {
		t.Fatal("expired URLs must be rejected")
	}
	other := NewSigner([]byte("a-different-secret-0123456789-abcdefgh"))
	if other.Verify("photo-1", "thumb", q["exp"], q["sig"]) {
		t.Fatal("another secret must not validate")
	}
}

func itoa(n int64) string {
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestLocalStorageRejectsTraversal(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../escape", "/etc/passwd", "", "a/../../escape"} {
		if err := l.Put(key, []byte("x")); err == nil {
			t.Errorf("key %q must be rejected", key)
		}
	}
	if err := l.Put("u/1.jpg", []byte("data")); err != nil {
		t.Fatal(err)
	}
	rc, err := l.Open("u/1.jpg")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "data" {
		t.Fatalf("got %q", b)
	}
	if err := l.Delete("u/1.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Open("u/1.jpg"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
