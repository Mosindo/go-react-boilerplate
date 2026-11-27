package storage

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestURLSignerRoundTripAndTampering(t *testing.T) {
	s := NewURLSigner([]byte("0123456789abcdef0123456789abcdef"))
	id := "7d8f3c1e-1111-4222-8333-444455556666"
	raw := s.PhotoURL(id)
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	exp, sig := u.Query().Get("exp"), u.Query().Get("sig")
	if !s.Verify(id, exp, sig) {
		t.Fatal("fresh signature must verify")
	}
	if s.Verify("7d8f3c1e-1111-4222-8333-444455556667", exp, sig) {
		t.Fatal("signature must be bound to the photo id")
	}
	if s.Verify(id, exp+"0", sig) {
		t.Fatal("signature must be bound to the expiry")
	}

	s.now = func() time.Time { return time.Now().Add(13 * time.Hour) }
	if s.Verify(id, exp, sig) {
		t.Fatal("expired signatures must be rejected")
	}
}

func TestURLSignerIsStableWithinBucket(t *testing.T) {
	s := NewURLSigner([]byte("0123456789abcdef0123456789abcdef"))
	base := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return base }
	a := s.PhotoURL("7d8f3c1e-1111-4222-8333-444455556666")
	s.now = func() time.Time { return base.Add(time.Hour) }
	b := s.PhotoURL("7d8f3c1e-1111-4222-8333-444455556666")
	if a != b {
		t.Fatal("URLs should be cache-stable within a bucket")
	}
}

func TestLocalStoreRejectsTraversal(t *testing.T) {
	store, err := NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, key := range []string{"../etc/passwd", "photos/../../x.jpg", "photos/abc/def.jpg"} {
		if err := store.Put(ctx, key, []byte("x")); err != ErrInvalidKey {
			t.Fatalf("expected ErrInvalidKey for %q, got %v", key, err)
		}
	}

	key, err := NewPhotoKey("7d8f3c1e-1111-4222-8333-444455556666")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, key, []byte("jpeg-bytes")); err != nil {
		t.Fatal(err)
	}
	f, _, err := store.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if !strings.EqualFold(string(data), "jpeg-bytes") {
		t.Fatal("unexpected content")
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Open(ctx, key); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
