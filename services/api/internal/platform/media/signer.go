package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

// URLTTL is the minimum validity of a signed photo URL. Expiries are aligned
// on hour boundaries so a URL stays identical (and cacheable) for a while.
const URLTTL = time.Hour

// Signer issues and verifies HMAC-signed photo URLs.
type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner derives a dedicated key so the JWT secret is never reused as-is.
func NewSigner(secret string) *Signer {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("lueur/media-url/v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

// PhotoURL returns a relative URL the client prefixes with the API base URL.
func (s *Signer) PhotoURL(photoID string) string {
	exp := s.now().Add(URLTTL).Truncate(time.Hour).Add(time.Hour).Unix()
	expStr := strconv.FormatInt(exp, 10)
	return "/media/photos/" + photoID + "?exp=" + expStr + "&sig=" + s.sign(photoID, expStr)
}

// Verify checks a signature and its expiry in constant time.
func (s *Signer) Verify(photoID, expStr, sig string) bool {
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return false
	}
	expected := s.sign(photoID, expStr)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *Signer) sign(photoID, exp string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID))
	mac.Write([]byte{'.'})
	mac.Write([]byte(exp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
