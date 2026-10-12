// Package mediaurl creates and verifies short-lived signed URLs for private media.
// Photos are never publicly listable: a URL is only handed to users who are
// allowed to see the owner's profile, and it expires.
package mediaurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const (
	// bucket keeps URLs stable for a window so that image caches keep working.
	bucket = 6 * time.Hour
	// Validity is between bucket and 2*bucket from the time of signing.
	validity = bucket
)

type Signer struct {
	key []byte
	now func() time.Time
}

// New derives a dedicated signing key from the application secret.
func New(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("media-url-v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

func (s *Signer) sign(photoID, size string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	fmt.Fprintf(mac, "%s|%s|%d", photoID, size, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

// Path returns a relative URL such as /media/photos/<id>?size=thumb&exp=..&sig=..
func (s *Signer) Path(photoID, size string) string {
	t := s.now().Unix()
	b := int64(bucket / time.Second)
	exp := (t/b)*b + b + int64(validity/time.Second)
	return fmt.Sprintf("/media/photos/%s?size=%s&exp=%d&sig=%s", photoID, size, exp, s.sign(photoID, size, exp))
}

// Verify checks signature and expiry.
func (s *Signer) Verify(photoID, size, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	want := s.sign(photoID, size, exp)
	return hmac.Equal([]byte(want), []byte(sig))
}
