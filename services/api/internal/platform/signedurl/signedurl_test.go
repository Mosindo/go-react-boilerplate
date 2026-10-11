package signedurl

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignVerify(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"))
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	raw := s.PhotoURL("photo-1", now)
	if raw != s.PhotoURL("photo-1", now.Add(5*time.Hour)) {
		t.Fatal("URL must be stable during the same UTC day")
	}
	u, _ := url.Parse(raw)
	q := u.Query()
	if !strings.HasPrefix(u.Path, "/photos/photo-1/") || !s.Verify("photo-1", q.Get("exp"), q.Get("sig"), now.Add(23*time.Hour)) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("photo-2", q.Get("exp"), q.Get("sig"), now) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", q.Get("exp"), q.Get("sig"), now.Add(72*time.Hour)) {
		t.Fatal("expired URL accepted")
	}
	if s.Verify("photo-1", q.Get("exp"), "00"+q.Get("sig")[2:], now) || s.Verify("photo-1", "abc", q.Get("sig"), now) {
		t.Fatal("tampered signature accepted")
	}
}
