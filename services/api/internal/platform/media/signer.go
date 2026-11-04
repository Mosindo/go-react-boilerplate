// Package media provides signed photo URLs, image processing and blob storage.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// urlWindow makes URLs stable for a window so HTTP/image caches keep working.
const urlWindow = 6 * time.Hour

// Signer issues and verifies expiring URLs for photo files. Photo bytes are only
// reachable through such a URL, and the API only hands them to authorised viewers.
type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner derives a dedicated key from the application secret.
func NewSigner(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("photo-url-signing-v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	fmt.Fprintf(mac, "%s|%d", photoID, exp)
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// URL returns a relative URL valid for between 6 and 12 hours.
func (s *Signer) URL(photoID string) string {
	window := int64(urlWindow / time.Second)
	exp := (s.now().Unix()/window + 2) * window
	return fmt.Sprintf("/photos/%s/file?exp=%d&sig=%s", photoID, exp, s.sign(photoID, exp))
}

// Verify checks the signature and expiry.
func (s *Signer) Verify(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || exp < s.now().Unix() {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(s.sign(photoID, exp)))
}
