// Package urlsign creates short-lived HMAC signatures for resource URLs.
// It lets image components load protected photos without sending the bearer
// token, while keeping photo URLs unguessable and expiring.
package urlsign

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
	bucket time.Duration
	now    func() time.Time
}

// New derives a dedicated signing key from secret so that photo signatures
// can never be confused with JWT signatures.
func New(secret []byte, ttl time.Duration) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("lueur/urlsign/v1"))
	return &Signer{key: mac.Sum(nil), ttl: ttl, bucket: time.Hour, now: time.Now}
}

// Sign returns the expiry and signature for resource. Expiries are aligned on
// an hourly bucket so the same URL is reused for a while, which keeps client
// image caches effective.
func (s *Signer) Sign(resource string) (int64, string) {
	exp := s.now().Add(s.ttl).Truncate(s.bucket).Add(s.bucket).Unix()
	return exp, s.signature(resource, exp)
}

// Query returns the `exp` and `sig` query string for resource.
func (s *Signer) Query(resource string) string {
	exp, sig := s.Sign(resource)
	return "exp=" + strconv.FormatInt(exp, 10) + "&sig=" + sig
}

func (s *Signer) Verify(resource, rawExp, sig string) bool {
	exp, err := strconv.ParseInt(rawExp, 10, 64)
	if err != nil || exp < s.now().Unix() {
		return false
	}
	expected := s.signature(resource, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *Signer) signature(resource string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(resource))
	mac.Write([]byte{0})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
