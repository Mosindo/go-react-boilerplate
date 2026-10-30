package signedurl

import (
	"strconv"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"))
	exp, sig := s.Query("photo-1")
	expRaw := strconv.FormatInt(exp, 10)

	if !s.Verify("photo-1", expRaw, sig) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("photo-2", expRaw, sig) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", strconv.FormatInt(exp+3600, 10), sig) {
		t.Fatal("signature must be bound to the expiry")
	}
	if s.Verify("photo-1", expRaw, sig[:len(sig)-1]+"0") {
		t.Fatal("tampered signature accepted")
	}

	now := time.Unix(exp+1, 0)
	s.now = func() time.Time { return now }
	if s.Verify("photo-1", expRaw, sig) {
		t.Fatal("expired signature accepted")
	}
}
