// Package media issues and verifies expiring signed URLs for private media.
// Photos are never served from a public bucket: the API only hands out URLs
// to users allowed to see a profile, and each URL expires.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// window controls URL stability: URLs stay identical within a window so
// clients can cache images, and remain valid for at least one full window.
const window = 6 * time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner derives a dedicated signing key from the application secret.
func NewSigner(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("lueur-media-signing-v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

func (s *Signer) signature(photoID string, expires int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID))
	mac.Write([]byte{0})
	mac.Write([]byte(strconv.FormatInt(expires, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// PhotoURL returns the relative, signed URL of a photo.
func (s *Signer) PhotoURL(photoID string) string {
	bucket := s.now().Unix() / int64(window.Seconds())
	expires := (bucket + 2) * int64(window.Seconds())
	return "/media/photos/" + photoID + "?exp=" + strconv.FormatInt(expires, 10) + "&sig=" + s.signature(photoID, expires)
}

// Verify checks the signature and expiry of a photo URL.
func (s *Signer) Verify(photoID, expiresRaw, signature string) bool {
	expires, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil || expires < s.now().Unix() {
		return false
	}
	expected := s.signature(photoID, expires)
	return hmac.Equal([]byte(expected), []byte(signature))
}
