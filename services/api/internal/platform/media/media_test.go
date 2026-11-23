package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	s.now = func() time.Time { return now }

	raw := s.Sign("photo-1")
	if !strings.HasPrefix(raw, "/media/photos/photo-1?") {
		t.Fatalf("unexpected url %s", raw)
	}
	u, _ := url.Parse(raw)
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
	now = now.Add(3 * time.Hour)
	if s.Verify("photo-1", exp, sig) {
		t.Fatal("expired signature must fail")
	}
}
