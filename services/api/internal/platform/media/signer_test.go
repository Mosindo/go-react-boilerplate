package media

import (
	"net/url"
	"strings"
	"testing"
)

func TestSignerRoundTripAndTamper(t *testing.T) {
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	raw := s.URL("11111111-1111-1111-1111-111111111111")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimPrefix(u.Path, "/media/")
	q := u.Query()
	if !s.Verify(id, q.Get("exp"), q.Get("sig")) {
		t.Fatal("expected valid signature")
	}
	if s.Verify("22222222-2222-2222-2222-222222222222", q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify(id, "1", q.Get("sig")) {
		t.Fatal("expired/modified exp must fail")
	}
	if s.URL("11111111-1111-1111-1111-111111111111") != raw {
		t.Fatal("URL should be stable inside a bucket")
	}
}
