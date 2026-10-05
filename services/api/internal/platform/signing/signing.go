// Package signing issues short-lived signed URLs for private media so images can be loaded by a plain
// <Image uri> (no auth header) while access stays bound to a viewer who was authorised when the URL was issued.
package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const urlBucket = time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

// New derives a dedicated key from the application secret.
func New(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("photo-url-v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	fmt.Fprintf(mac, "%s|%d", photoID, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

// PhotoPath returns an API-relative URL valid for 1 to 2 hours. The expiry is bucketed per hour so the
// same URL is returned repeatedly, which keeps client image caches effective.
func (s *Signer) PhotoPath(photoID string) string {
	exp := s.now().Truncate(urlBucket).Add(2 * urlBucket).Unix()
	return fmt.Sprintf("/photos/%s/image?exp=%d&sig=%s", photoID, exp, s.sign(photoID, exp))
}

// VerifyPhoto checks signature and expiry in constant time.
func (s *Signer) VerifyPhoto(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(s.sign(photoID, exp)), []byte(sig))
}
