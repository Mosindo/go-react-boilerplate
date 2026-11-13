package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignerRoundTrip(t *testing.T) {
	current := time.Unix(1_700_000_000, 0)
	s := NewSigner([]byte("a-very-long-secret-used-for-testing-only"), time.Hour)
	s.now = func() time.Time { return current }

	raw := s.PhotoURL("photo-1")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u.Path, "/media/photos/photo-1") {
		t.Fatalf("unexpected path %s", u.Path)
	}
	q := u.Query()
	if !s.Verify("photo-1", q.Get("exp"), q.Get("sig")) {
		t.Fatal("expected valid signature")
	}
	if s.Verify("photo-2", q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", q.Get("exp")+"0", q.Get("sig")) {
		t.Fatal("tampered expiry must fail")
	}

	current = current.Add(3 * time.Hour)
	if s.Verify("photo-1", q.Get("exp"), q.Get("sig")) {
		t.Fatal("expired signature must fail")
	}
}
