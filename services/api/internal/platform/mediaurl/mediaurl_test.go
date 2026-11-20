package mediaurl

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPhotoURLRoundTrip(t *testing.T) {
	signer := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	raw := signer.PhotoURL("photo-1")
	if !strings.HasPrefix(raw, "/media/photos/photo-1?") {
		t.Fatalf("unexpected url %q", raw)
	}
	parsed, _ := url.Parse(raw)
	exp, sig := parsed.Query().Get("exp"), parsed.Query().Get("sig")
	if !signer.Verify("photo-1", exp, sig) {
		t.Fatal("expected signature to verify")
	}
	if signer.Verify("photo-2", exp, sig) {
		t.Fatal("signature must be bound to the photo id")
	}
	if signer.Verify("photo-1", exp, sig[:len(sig)-1]+"0") && !strings.HasSuffix(sig, "0") {
		t.Fatal("tampered signature must fail")
	}

	signer.now = func() time.Time { return time.Now().Add(48 * time.Hour) }
	if signer.Verify("photo-1", exp, sig) {
		t.Fatal("expired url must fail")
	}
}

func TestPhotoURLStableWithinBucket(t *testing.T) {
	signer := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	base := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	signer.now = func() time.Time { return base }
	first := signer.PhotoURL("p")
	signer.now = func() time.Time { return base.Add(2 * time.Hour) }
	if first != signer.PhotoURL("p") {
		t.Fatal("expected identical URL inside the same bucket")
	}
}
