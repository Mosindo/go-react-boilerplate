package urlsign

import (
	"strconv"
	"testing"
	"time"
)

func TestSignVerify(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"), time.Hour)
	exp, sig := s.Sign("photo-1")
	raw := strconv.FormatInt(exp, 10)
	if !s.Verify("photo-1", raw, sig) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("photo-2", raw, sig) {
		t.Fatal("signature must be bound to the resource")
	}
	if s.Verify("photo-1", strconv.FormatInt(exp+3600, 10), sig) {
		t.Fatal("signature must be bound to the expiry")
	}
	if s.Verify("photo-1", raw, sig+"x") {
		t.Fatal("tampered signature accepted")
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"), time.Hour)
	exp, sig := s.Sign("photo-1")
	s.now = func() time.Time { return time.Unix(exp+1, 0) }
	if s.Verify("photo-1", strconv.FormatInt(exp, 10), sig) {
		t.Fatal("expired signature accepted")
	}
}
