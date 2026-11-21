package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func parse(t *testing.T, raw string) url.Values {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return u.Query()
}

func TestSignerRoundTripAndTamper(t *testing.T) {
	signer := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	raw := signer.URL("photo-1", VariantThumb)
	if !strings.HasPrefix(raw, "/media/photos/photo-1?") {
		t.Fatalf("unexpected url %s", raw)
	}
	q := parse(t, raw)
	if !signer.Verify("photo-1", VariantThumb, q.Get("exp"), q.Get("sig")) {
		t.Fatal("expected signature to verify")
	}
	if signer.Verify("photo-2", VariantThumb, q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to photo id")
	}
	if signer.Verify("photo-1", VariantFull, q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to variant")
	}
	if signer.Verify("photo-1", VariantThumb, "99999999999", q.Get("sig")) {
		t.Fatal("signature must be bound to expiry")
	}
}

func TestSignerExpires(t *testing.T) {
	signer := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	q := parse(t, signer.URL("photo-1", VariantFull))
	signer.now = func() time.Time { return time.Now().Add(3 * time.Hour) }
	if signer.Verify("photo-1", VariantFull, q.Get("exp"), q.Get("sig")) {
		t.Fatal("expected expired url to be rejected")
	}
}
