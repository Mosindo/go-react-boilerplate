// Package media stores processed photos on disk and issues short-lived signed URLs
// so images can be loaded by <Image> components without exposing bearer tokens.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

// URLValidity is the minimum lifetime of an issued URL. Expirations are aligned on
// this window so the same photo keeps the same URL for a while (client cache friendly).
const URLValidity = time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner derives a dedicated media key from the application secret.
func NewSigner(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("media-url-signing-v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

// URL returns the relative signed URL for a storage key.
func (s *Signer) URL(storageKey string) string {
	window := int64(URLValidity.Seconds())
	exp := (s.now().Unix()/window + 2) * window
	q := url.Values{}
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", s.sign(storageKey, exp))
	return "/media/" + url.PathEscape(storageKey) + "?" + q.Encode()
}

// Verify checks a signature and expiration.
func (s *Signer) Verify(storageKey, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	expected := s.sign(storageKey, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *Signer) sign(storageKey string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(storageKey))
	mac.Write([]byte{'|'})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}
