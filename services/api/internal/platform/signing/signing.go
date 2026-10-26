// Package signing issues short-lived, tamper-proof URLs for private media so
// that image components can fetch photos without sending auth headers.
package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

// Window is the validity bucket. URLs are stable inside a bucket (cache friendly)
// and valid for between Window and 2*Window.
const Window = 6 * time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

func New(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("media-url-signing"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

func (s *Signer) sign(id string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(id + "|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// PhotoURL returns the relative URL where the photo can be fetched.
func (s *Signer) PhotoURL(photoID string) string {
	exp := (s.now().Unix()/int64(Window.Seconds()) + 2) * int64(Window.Seconds())
	q := url.Values{}
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", s.sign(photoID, exp))
	return "/photos/" + photoID + "/file?" + q.Encode()
}

// Verify checks signature and expiry.
func (s *Signer) Verify(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || exp < s.now().Unix() {
		return false
	}
	return hmac.Equal([]byte(s.sign(photoID, exp)), []byte(sig))
}
