package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignerRoundTripAndExpiry(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	s := NewSigner([]byte("a-very-long-secret-used-only-for-tests-123"))
	s.now = func() time.Time { return current }

	raw := s.PhotoURL("photo-1")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(parsed.Path, "/media/photos/photo-1") {
		t.Fatalf("unexpected path %s", parsed.Path)
	}
	exp, sig := parsed.Query().Get("exp"), parsed.Query().Get("sig")
	if !s.Verify("photo-1", exp, sig) {
		t.Fatal("expected valid signature")
	}
	if s.Verify("photo-2", exp, sig) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", exp, strings.Repeat("0", len(sig))) {
		t.Fatal("tampered signature must fail")
	}
	if raw != s.PhotoURL("photo-1") {
		t.Fatal("URLs must be stable within a window for caching")
	}

	current = current.Add(3 * time.Hour)
	if s.Verify("photo-1", exp, sig) {
		t.Fatal("expired URL must fail")
	}
}
