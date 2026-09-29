package urlsign

import (
	"testing"
	"time"
)

func TestSignVerify(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"))
	exp, sig := s.Sign("photo-1")
	if !s.Verify("photo-1", exp, sig) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("photo-2", exp, sig) {
		t.Fatal("signature must be bound to the resource")
	}
	if s.Verify("photo-1", exp+1, sig) {
		t.Fatal("signature must be bound to the expiry")
	}
	if s.Verify("photo-1", exp, sig[:len(sig)-1]+"0") && sig[len(sig)-1] != '0' {
		t.Fatal("tampered signature accepted")
	}
	s.now = func() time.Time { return time.Unix(exp+1, 0) }
	if s.Verify("photo-1", exp, sig) {
		t.Fatal("expired signature accepted")
	}
}

func TestSignIsStableWithinWindow(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"))
	base := time.Unix(1_700_000_000, 0)
	s.now = func() time.Time { return base }
	e1, s1 := s.Sign("x")
	s.now = func() time.Time { return base.Add(time.Minute) }
	e2, s2 := s.Sign("x")
	if e1 != e2 || s1 != s2 {
		t.Fatal("URLs should be cache-stable inside a window")
	}
}
