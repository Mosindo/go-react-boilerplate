// Package signedurl issues short-lived, tamper-proof links to private media so
// that plain image components can load them without an Authorization header.
package signedurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const (
	ttlBucket = time.Hour
	validity  = 2 * ttlBucket
)

type Signer struct {
	key []byte
	now func() time.Time
}

// New derives a dedicated key from the JWT secret so the two uses never overlap.
func New(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("signedurl/v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

// Query returns the expiry and signature to append to a media URL. The
// expiry is rounded to the hour so URLs stay stable (and cacheable) within a
// bucket and remain valid for one to two hours.
func (s *Signer) Query(id string) (exp int64, sig string) {
	bucket := s.now().Truncate(ttlBucket)
	exp = bucket.Add(validity).Unix()
	return exp, s.sign(id, exp)
}

func (s *Signer) Verify(id, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(s.sign(id, exp)), []byte(sig))
}

func (s *Signer) sign(id string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	fmt.Fprintf(mac, "%s|%d", id, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

// PhotoPath returns the relative, signed URL of a photo's image file.
func (s *Signer) PhotoPath(id string) string {
	exp, sig := s.Query(id)
	return fmt.Sprintf("/photos/%s/file?exp=%d&sig=%s", id, exp, sig)
}
