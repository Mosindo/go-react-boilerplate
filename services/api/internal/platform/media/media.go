// Package media builds and verifies signed, expiring URLs for stored photos.
//
// Photos are only reachable through URLs that the API hands out in responses
// that already passed authorization checks (own profile, discovery, matches).
// Expirations are aligned to fixed windows so a given photo keeps the same URL
// for several hours, which lets clients cache images efficiently.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

const window = 6 * time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner derives a dedicated signing key from the application secret so the
// JWT key is never used directly for another purpose.
func NewSigner(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("media-url-signing-v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

// URL returns a relative URL such as /media/<key>?exp=...&sig=...
func (s *Signer) URL(storageKey string) string {
	now := s.now().UTC()
	exp := now.Truncate(window).Add(2 * window).Unix()
	expStr := strconv.FormatInt(exp, 10)
	return "/media/" + storageKey + "?exp=" + expStr + "&sig=" + s.sign(storageKey, expStr)
}

// Verify checks the signature and that the URL has not expired.
func (s *Signer) Verify(storageKey, expStr, sig string) bool {
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	expected := s.sign(storageKey, expStr)
	return hmac.Equal([]byte(expected), []byte(sig))
}

// MaxAge returns how long a response for this URL may be cached.
func (s *Signer) MaxAge(expStr string) time.Duration {
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return 0
	}
	d := time.Until(time.Unix(exp, 0))
	if d < 0 {
		return 0
	}
	return d
}

func (s *Signer) sign(storageKey, exp string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(storageKey))
	mac.Write([]byte{0})
	mac.Write([]byte(exp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
