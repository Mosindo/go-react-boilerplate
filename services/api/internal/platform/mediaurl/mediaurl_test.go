package mediaurl

import (
	"net/url"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"))
	raw := s.URL("photo-1")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if !s.Verify("photo-1", q.Get("exp"), q.Get("sig")) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("photo-2", q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", q.Get("exp"), "deadbeef") {
		t.Fatal("forged signature accepted")
	}
	if s.Verify("photo-1", "abc", q.Get("sig")) {
		t.Fatal("malformed expiry accepted")
	}
}

func TestExpiryAndStability(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"))
	base := time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC)
	s.now = func() time.Time { return base }
	first := s.URL("p")
	s.now = func() time.Time { return base.Add(20 * time.Minute) }
	if s.URL("p") != first {
		t.Fatal("URL must be stable inside a time bucket")
	}

	u, _ := url.Parse(first)
	q := u.Query()
	s.now = func() time.Time { return base.Add(3 * time.Hour) }
	if s.Verify("p", q.Get("exp"), q.Get("sig")) {
		t.Fatal("expired URL accepted")
	}
}
