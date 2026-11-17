package media

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPhotoURLSignAndVerify(t *testing.T) {
	signer := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	now := time.Unix(1_700_000_000, 0)
	signer.now = func() time.Time { return now }

	raw := signer.PhotoURL("photo-1")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if !strings.HasPrefix(parsed.Path, "/media/photos/photo-1") {
		t.Fatalf("unexpected path %q", parsed.Path)
	}
	exp, sig := parsed.Query().Get("exp"), parsed.Query().Get("sig")
	if !signer.Verify("photo-1", exp, sig) {
		t.Fatal("expected signature to verify")
	}
	if signer.Verify("photo-2", exp, sig) {
		t.Fatal("signature must be bound to the photo id")
	}
	if signer.Verify("photo-1", exp, strings.Repeat("0", len(sig))) {
		t.Fatal("tampered signature must fail")
	}
	if raw != signer.PhotoURL("photo-1") {
		t.Fatal("URL should be stable within a window")
	}

	now = now.Add(3 * Window)
	if signer.Verify("photo-1", exp, sig) {
		t.Fatal("expired URL must fail")
	}
}
