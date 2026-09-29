// Package urlsign issues and verifies short-lived HMAC-signed tokens for private media URLs.
// Access rules are evaluated when a URL is issued; the signature only proves the URL was
// issued by the API for that exact resource and has not expired.
package urlsign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// window makes signed URLs stable for a few hours so clients can cache images.
const window = 6 * time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

// New derives a dedicated key from the JWT secret so the two uses never share raw key material.
func New(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("urlsign/v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

// Sign returns (expiresUnix, signature) for a resource id. The expiry is aligned on window
// boundaries: the URL is valid for between one and two windows.
func (s *Signer) Sign(resourceID string) (int64, string) {
	exp := (s.now().Unix()/int64(window.Seconds()) + 2) * int64(window.Seconds())
	return exp, s.mac(resourceID, exp)
}

func (s *Signer) Verify(resourceID string, exp int64, signature string) bool {
	if exp < s.now().Unix() {
		return false
	}
	return hmac.Equal([]byte(signature), []byte(s.mac(resourceID, exp)))
}

func (s *Signer) mac(resourceID string, exp int64) string {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte(resourceID))
	m.Write([]byte{0})
	m.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(m.Sum(nil))
}
