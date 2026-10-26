package signing

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPhotoURLVerifyAndTamper(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"))
	raw := s.PhotoURL("photo-1")
	u, err := url.Parse(raw)
	if err != nil || !strings.HasPrefix(u.Path, "/photos/photo-1/file") {
		t.Fatalf("unexpected url %q", raw)
	}
	q := u.Query()
	if !s.Verify("photo-1", q.Get("exp"), q.Get("sig")) {
		t.Fatal("fresh signature must verify")
	}
	if s.Verify("photo-2", q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", "9999999999", q.Get("sig")) {
		t.Fatal("signature must be bound to expiry")
	}
	s.now = func() time.Time { return time.Now().Add(3 * Window) }
	if s.Verify("photo-1", q.Get("exp"), q.Get("sig")) {
		t.Fatal("expired signature must be rejected")
	}
	if New([]byte("another-secret-another-secret-123")).Verify("photo-1", q.Get("exp"), q.Get("sig")) {
		t.Fatal("different secret must not verify")
	}
}
