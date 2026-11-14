// Package media issues short-lived signed URLs for private files (photos).
// Access control happens when a URL is issued: the API only returns photo
// URLs inside responses the viewer is allowed to see. The signature makes
// the URL unguessable and time-bound, and lets image components load it
// without custom headers (works with native and web image caches).
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

const bucket = time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner derives a dedicated signing key from the server secret so photo
// signatures can never be confused with JWTs.
func NewSigner(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("lueur/media-url/v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID))
	mac.Write([]byte{0})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// PhotoPath returns a relative signed path. Expiry is rounded to hour
// buckets (valid 1-2h) so URLs stay stable and cacheable on clients.
func (s *Signer) PhotoPath(photoID string) string {
	exp := s.now().Truncate(bucket).Add(2 * bucket).Unix()
	return "/media/photos/" + photoID + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.sign(photoID, exp)
}

// Verify checks signature and expiry in constant time.
func (s *Signer) Verify(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	expected := s.sign(photoID, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}
