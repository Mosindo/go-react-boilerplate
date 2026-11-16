package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignerRoundTrip(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"), time.Hour)
	s.now = func() time.Time { return now }

	raw := s.URL("photo-1")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u.Path, "/media/photos/photo-1") {
		t.Fatalf("unexpected path %s", u.Path)
	}
	exp, sig := u.Query().Get("exp"), u.Query().Get("sig")
	if !s.Verify("photo-1", exp, sig) {
		t.Fatal("expected signature to verify")
	}
	if s.Verify("photo-2", exp, sig) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", exp, sig+"x") {
		t.Fatal("tampered signature must fail")
	}
	if s.URL("photo-1") != raw {
		t.Fatal("URLs should be stable within a bucket")
	}

	now = now.Add(2 * time.Hour)
	if s.Verify("photo-1", exp, sig) {
		t.Fatal("expired signature must fail")
	}
}
