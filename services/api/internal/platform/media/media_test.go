package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignedURLRoundTripAndTamper(t *testing.T) {
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	key := "0123456789abcdef0123456789abcdef.jpg"
	raw := s.URL(key)
	if !strings.HasPrefix(raw, "/media/"+key+"?") {
		t.Fatalf("unexpected url %s", raw)
	}
	u, _ := url.Parse(raw)
	exp, sig := u.Query().Get("exp"), u.Query().Get("sig")
	if !s.Verify(key, exp, sig) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("ffffffffffffffffffffffffffffffff.jpg", exp, sig) {
		t.Fatal("signature reused for another key")
	}
	if s.Verify(key, exp+"0", sig) {
		t.Fatal("tampered expiration accepted")
	}
	if s.URL(key) != raw {
		t.Fatal("url should be stable inside a window")
	}

	s.now = func() time.Time { return time.Now().Add(13 * time.Hour) }
	if s.Verify(key, exp, sig) {
		t.Fatal("expired url accepted")
	}
}
