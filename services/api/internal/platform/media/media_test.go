package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignerRoundTrip(t *testing.T) {
	now := time.Unix(1_700_000_123, 0)
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	s.now = func() time.Time { return now }

	raw := s.URL("abc.jpg")
	if !strings.HasPrefix(raw, "/media/abc.jpg?") {
		t.Fatalf("unexpected url %q", raw)
	}
	u, _ := url.Parse(raw)
	exp, sig := u.Query().Get("exp"), u.Query().Get("sig")
	if !s.Verify("abc.jpg", exp, sig) {
		t.Fatal("expected signature to verify")
	}
	if s.Verify("other.jpg", exp, sig) {
		t.Fatal("signature must be bound to the key")
	}
	if s.Verify("abc.jpg", exp+"0", sig) {
		t.Fatal("signature must be bound to the expiry")
	}
	if raw != s.URL("abc.jpg") {
		t.Fatal("URL must be stable within the window")
	}

	now = now.Add(3 * time.Hour)
	if s.Verify("abc.jpg", exp, sig) {
		t.Fatal("expired signature must be rejected")
	}

	other := NewSigner([]byte("another-secret-another-secret-xx"))
	if other.Verify("abc.jpg", exp, sig) {
		t.Fatal("signature must depend on the secret")
	}
}
