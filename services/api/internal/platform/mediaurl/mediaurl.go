// Package mediaurl issues and verifies short-lived signed URLs for private media.
// URLs are stable within a time bucket so HTTP caches keep working, and they expire
// after at most two buckets.
package mediaurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const bucket = time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

// New derives a dedicated signing key from the application secret so the JWT secret
// itself is never used directly for URL signatures.
func New(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("mediaurl.v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

// URL returns the relative signed URL for a photo.
func (s *Signer) URL(photoID string) string {
	exp := s.now().Truncate(bucket).Add(2 * bucket).Unix()
	return fmt.Sprintf("/media/photos/%s?exp=%d&sig=%s", photoID, exp, s.sign(photoID, exp))
}

// Verify checks signature and expiry.
func (s *Signer) Verify(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(s.sign(photoID, exp)))
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	fmt.Fprintf(mac, "%s|%d", photoID, exp)
	return hex.EncodeToString(mac.Sum(nil))
}
