package mediaurl

import (
	"net/url"
	"testing"
	"time"
)

func TestSignVerifyAndExpiry(t *testing.T) {
	s := New([]byte("0123456789abcdef0123456789abcdef"))
	base := time.Unix(1_700_000_000, 0)
	s.now = func() time.Time { return base }

	u, err := url.Parse(s.Path("photo-1", "full"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if !s.Verify("photo-1", "full", q.Get("exp"), q.Get("sig")) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("photo-2", "full", q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify("photo-1", "thumb", q.Get("exp"), q.Get("sig")) {
		t.Fatal("signature must be bound to the size")
	}
	if s.Path("photo-1", "full") != u.String() {
		t.Fatal("URL should be stable within a time bucket")
	}
	s.now = func() time.Time { return base.Add(13 * time.Hour) }
	if s.Verify("photo-1", "full", q.Get("exp"), q.Get("sig")) {
		t.Fatal("expired URL accepted")
	}
}
