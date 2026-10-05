package signing

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func parts(path string) (exp, sig string) {
	q := path[strings.Index(path, "?")+1:]
	for _, kv := range strings.Split(q, "&") {
		switch {
		case strings.HasPrefix(kv, "exp="):
			exp = kv[4:]
		case strings.HasPrefix(kv, "sig="):
			sig = kv[4:]
		}
	}
	return
}

func TestPhotoURLVerification(t *testing.T) {
	s := New([]byte("secret-secret-secret-secret-secret"))
	base := time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC)
	s.now = func() time.Time { return base }

	path := s.PhotoPath("photo-1")
	exp, sig := parts(path)
	if !s.VerifyPhoto("photo-1", exp, sig) {
		t.Fatal("fresh URL must verify")
	}
	if s.VerifyPhoto("photo-2", exp, sig) {
		t.Fatal("signature is bound to the photo id")
	}
	if s.VerifyPhoto("photo-1", exp, sig[:len(sig)-1]+"0") && sig[len(sig)-1] != '0' {
		t.Fatal("tampered signature must fail")
	}
	later, _ := strconv.ParseInt(exp, 10, 64)
	if s.VerifyPhoto("photo-1", strconv.FormatInt(later+3600, 10), sig) {
		t.Fatal("extending the expiry must invalidate the signature")
	}
	if s.VerifyPhoto("photo-1", "abc", sig) || s.VerifyPhoto("photo-1", "", "") {
		t.Fatal("malformed input must fail")
	}

	// Stable within the hour (client image caches keep working), valid between one and two hours.
	s.now = func() time.Time { return base.Add(20 * time.Minute) }
	if s.PhotoPath("photo-1") != path {
		t.Fatal("URL must be stable within the same hour")
	}
	s.now = func() time.Time { return base.Add(80 * time.Minute) }
	if !s.VerifyPhoto("photo-1", exp, sig) {
		t.Fatal("URL must still be valid after 80 minutes")
	}
	s.now = func() time.Time { return base.Add(100 * time.Minute) }
	if s.VerifyPhoto("photo-1", exp, sig) {
		t.Fatal("URL must expire")
	}

	other := New([]byte("a-different-secret-a-different-one"))
	other.now = func() time.Time { return base }
	if other.VerifyPhoto("photo-1", exp, sig) {
		t.Fatal("another secret must not verify")
	}
}
