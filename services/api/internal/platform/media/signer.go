// Package media issues and verifies short-lived signed URLs for private media.
// Photos are never publicly listable: the API only hands signed URLs to users
// allowed to see a profile, and the URL expires.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type Signer struct {
	key    []byte
	ttl    time.Duration
	bucket time.Duration
	now    func() time.Time
}

// NewSigner derives a dedicated signing key from the application secret so a
// leaked media signature can never be used as a JWT signature and vice versa.
func NewSigner(appSecret []byte, ttl time.Duration) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("media-url-signing-v1"))
	return &Signer{key: mac.Sum(nil), ttl: ttl, bucket: 15 * time.Minute, now: time.Now}
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	fmt.Fprintf(mac, "%s|%d", photoID, exp)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// URL returns a relative signed path. Expiry is rounded up to a bucket so the
// same photo gets a stable URL for a while, which keeps client image caches hot.
func (s *Signer) URL(photoID string) string {
	exp := s.now().Add(s.ttl).Truncate(s.bucket).Add(s.bucket).Unix()
	q := url.Values{}
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", s.sign(photoID, exp))
	return "/media/photos/" + photoID + "?" + q.Encode()
}

// Verify checks a signature and expiry.
func (s *Signer) Verify(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	expected := s.sign(photoID, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}
