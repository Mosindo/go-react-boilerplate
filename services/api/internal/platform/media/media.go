// Package media issues and verifies short-lived signed URLs for user photos.
//
// Photos are never publicly listable: a URL is only handed out to a viewer the
// API already authorized to see the profile, and it expires. Expiry is aligned
// on hourly windows so a given photo keeps the same URL for about an hour,
// which keeps client/CDN caching effective.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

const window = time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner derives a dedicated signing key from the application secret so
// media signatures can never be confused with JWT signatures.
func NewSigner(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("lueur/media-url/v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

// URL returns a relative URL (clients prefix it with the API base URL).
func (s *Signer) URL(storageKey string) string {
	expires := s.now().Truncate(window).Add(2 * window).Unix()
	exp := strconv.FormatInt(expires, 10)
	return "/media/" + storageKey + "?exp=" + exp + "&sig=" + s.sign(storageKey, exp)
}

// Verify checks signature and expiry.
func (s *Signer) Verify(storageKey, exp, sig string) bool {
	expires, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || s.now().Unix() > expires {
		return false
	}
	expected := s.sign(storageKey, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *Signer) sign(storageKey, exp string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(storageKey))
	mac.Write([]byte{0})
	mac.Write([]byte(exp))
	return hex.EncodeToString(mac.Sum(nil))
}
