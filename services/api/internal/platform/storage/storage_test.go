package storage

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestURLSignerRoundTripAndExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	s := NewURLSigner([]byte("0123456789abcdef0123456789abcdef"), time.Hour)
	s.now = func() time.Time { return now }

	key := strings.Repeat("a", 32) + ".jpg"
	signed := s.Sign(key)
	u, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if !s.Verify(key, q.Get("exp"), q.Get("sig")) {
		t.Fatal("expected valid signature")
	}
	if s.Verify(strings.Repeat("b", 32)+".jpg", q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to the key")
	}
	if s.Verify(key, "9999999999", q.Get("sig")) {
		t.Fatal("signature must be bound to the expiry")
	}
	if s.Sign(key) != signed {
		t.Fatal("URL should be stable within a window")
	}

	now = now.Add(3 * time.Hour)
	if s.Verify(key, q.Get("exp"), q.Get("sig")) {
		t.Fatal("expired URL must be rejected")
	}
}

func TestLocalStoreRejectsUnsafeKeys(t *testing.T) {
	st, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../etc/passwd", "abc.jpg", strings.Repeat("a", 32) + ".png", ""} {
		if err := st.Put(context.Background(), key, []byte("x")); err == nil {
			t.Fatalf("key %q should be rejected", key)
		}
	}

	key, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Put(context.Background(), key, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	f, _, err := st.Open(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	_ = f.Close()
	if string(data) != "hello" {
		t.Fatalf("unexpected content %q", data)
	}
	if err := st.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.Open(context.Background(), key); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
