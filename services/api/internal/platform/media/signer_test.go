package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignedURLRoundTripAndTampering(t *testing.T) {
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	now := time.Unix(1_700_000_000, 0)
	s.now = func() time.Time { return now }

	raw := s.PhotoURL("11111111-1111-1111-1111-111111111111")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimPrefix(u.Path, "/media/photos/")
	exp, sig := u.Query().Get("exp"), u.Query().Get("sig")

	if !s.Verify(id, exp, sig) {
		t.Fatal("expected valid signature")
	}
	if s.Verify("22222222-2222-2222-2222-222222222222", exp, sig) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify(id, exp+"0", sig) {
		t.Fatal("signature must be bound to the expiry")
	}
	now = now.Add(3 * time.Hour)
	if s.Verify(id, exp, sig) {
		t.Fatal("expired URL must be rejected")
	}
}

func TestSignedURLIsStableWithinWindow(t *testing.T) {
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	base := time.Unix(1_700_000_000, 0).Truncate(time.Hour)
	s.now = func() time.Time { return base.Add(time.Minute) }
	first := s.PhotoURL("id")
	s.now = func() time.Time { return base.Add(50 * time.Minute) }
	if second := s.PhotoURL("id"); second != first {
		t.Fatal("URL should be stable within a window to keep client caches warm")
	}
}
