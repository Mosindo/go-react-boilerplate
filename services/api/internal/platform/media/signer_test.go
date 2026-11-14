package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignerRoundTripAndTamper(t *testing.T) {
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	base := time.Date(2026, 1, 1, 10, 15, 0, 0, time.UTC)
	s.now = func() time.Time { return base }
	path := s.PhotoPath("11111111-1111-1111-1111-111111111111")
	u, err := url.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimPrefix(u.Path, "/media/photos/")
	q := u.Query()
	if !s.Verify(id, q.Get("exp"), q.Get("sig")) {
		t.Fatal("expected valid signature")
	}
	if s.Verify("22222222-2222-2222-2222-222222222222", q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify(id, q.Get("exp")+"0", q.Get("sig")) {
		t.Fatal("signature must be bound to the expiry")
	}
	if s.PhotoPath(id) != path {
		t.Fatal("paths must be stable within the same hour bucket")
	}
	s.now = func() time.Time { return base.Add(3 * time.Hour) }
	if s.Verify(id, q.Get("exp"), q.Get("sig")) {
		t.Fatal("expired signature must be rejected")
	}
}
