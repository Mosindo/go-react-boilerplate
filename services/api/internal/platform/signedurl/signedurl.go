// Package signedurl issues stable, expiring, HMAC-signed URLs for private photo content.
package signedurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const day = 24 * 3600

type Signer struct{ key []byte }

// New derives a dedicated signing key from the application secret.
func New(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("photo-url-signing-v1"))
	return &Signer{key: mac.Sum(nil)}
}

func (s *Signer) sign(id string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	fmt.Fprintf(mac, "%s.%d", id, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

// PhotoURL returns a relative URL valid for 24-48h. The expiry is day-aligned so the URL
// stays identical during a UTC day, which keeps client image caches effective.
func (s *Signer) PhotoURL(id string, now time.Time) string {
	exp := (now.Unix()/day + 2) * day
	return fmt.Sprintf("/photos/%s/content?exp=%d&sig=%s", id, exp, s.sign(id, exp))
}

func (s *Signer) Verify(id, expRaw, sig string, now time.Time) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || exp < now.Unix() {
		return false
	}
	return hmac.Equal([]byte(s.sign(id, exp)), []byte(sig))
}
