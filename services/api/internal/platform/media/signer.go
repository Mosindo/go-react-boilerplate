// Package media signs short-lived URLs for private media. Photos are never
// publicly listable: the API only hands signed URLs to viewers that are
// allowed to see a profile, and the URL expires on its own.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// Signer issues and verifies expiring HMAC signatures for media ids.
type Signer struct {
	key    []byte
	bucket time.Duration
	now    func() time.Time
}

// NewSigner derives a dedicated media key from the application secret so the
// JWT key is never used directly for another purpose.
func NewSigner(secret []byte, bucket time.Duration) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("media-url-signing-v1"))
	if bucket <= 0 {
		bucket = time.Hour
	}
	return &Signer{key: mac.Sum(nil), bucket: bucket, now: time.Now}
}

// PhotoURL returns a relative URL valid for between one and two buckets.
// Expiry is aligned on bucket boundaries so URLs stay stable (and cacheable
// by the client) within a bucket.
func (s *Signer) PhotoURL(photoID string) string {
	now := s.now().Unix()
	b := int64(s.bucket.Seconds())
	exp := (now/b + 2) * b
	return "/media/photos/" + photoID + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.sign(photoID, exp)
}

// Verify checks the signature and expiry for a media id.
func (s *Signer) Verify(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || exp < s.now().Unix() {
		return false
	}
	expected := s.sign(photoID, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID))
	mac.Write([]byte{'|'})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}
