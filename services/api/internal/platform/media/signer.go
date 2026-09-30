// Package media issues and verifies short-lived signed URLs for private files.
//
// Photos are never served from a public folder. Each URL embeds an expiry and
// an HMAC signature bound to the photo id, so a URL is only usable for a
// limited time and cannot be forged or re-targeted to another photo. Expiry is
// aligned on fixed windows so the same URL is reused for a while, which keeps
// client-side image caching effective.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

const (
	defaultWindow = time.Hour
)

type Signer struct {
	secret []byte
	window time.Duration
	now    func() time.Time
}

func NewSigner(secret []byte) *Signer {
	derived := hmac.New(sha256.New, secret)
	derived.Write([]byte("media-url-signing-v1"))
	return &Signer{secret: derived.Sum(nil), window: defaultWindow, now: time.Now}
}

// PhotoURL returns a relative URL valid for at least one window.
func (s *Signer) PhotoURL(photoID string) string {
	now := s.now().Unix()
	windowSeconds := int64(s.window.Seconds())
	expires := (now/windowSeconds + 2) * windowSeconds
	exp := strconv.FormatInt(expires, 10)
	return "/media/photos/" + photoID + "?exp=" + exp + "&sig=" + s.sign(photoID, exp)
}

// Verify checks a photo URL signature and expiry.
func (s *Signer) Verify(photoID, exp, sig string) bool {
	expires, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || expires < s.now().Unix() {
		return false
	}
	expected := s.sign(photoID, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *Signer) sign(photoID, exp string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(photoID))
	mac.Write([]byte{0})
	mac.Write([]byte(exp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
