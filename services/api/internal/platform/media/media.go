// Package media signs short-lived URLs for protected user photos so that
// images can be fetched by <Image> components without bearer headers while
// remaining unguessable and expiring.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

type Signer struct {
	key    []byte
	ttl    time.Duration
	window time.Duration
	now    func() time.Time
}

// NewSigner derives a dedicated signing key from the application secret so a
// media signature can never be replayed as a JWT signature.
func NewSigner(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("media-url-signing-v1"))
	return &Signer{key: mac.Sum(nil), ttl: 2 * time.Hour, window: time.Hour, now: time.Now}
}

// Sign returns a relative URL for photoID. Expiry is aligned on an hourly
// window so URLs stay stable (and cacheable) for at least an hour.
func (s *Signer) Sign(photoID string) string {
	exp := s.now().Truncate(s.window).Add(s.ttl).Unix()
	expStr := strconv.FormatInt(exp, 10)
	return "/media/photos/" + photoID + "?exp=" + expStr + "&sig=" + s.signature(photoID, expStr)
}

func (s *Signer) Verify(photoID, exp, sig string) bool {
	expUnix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || s.now().Unix() > expUnix {
		return false
	}
	expected := s.signature(photoID, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *Signer) signature(photoID, exp string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID))
	mac.Write([]byte{0})
	mac.Write([]byte(exp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
