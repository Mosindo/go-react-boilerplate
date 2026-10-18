package photos

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strconv"
	"time"
)

const (
	variantFull  = "full"
	variantThumb = "thumb"
)

// Signer issues expiring URLs for photo files. Authorization happens when a URL
// is handed out (only to users allowed to see that profile); the signature makes
// the URL unforgeable and unguessable, so images work in plain <img> tags.
type Signer struct{ secret []byte }

func NewSigner(secret []byte) *Signer { return &Signer{secret: secret} }

func (s *Signer) sign(photoID, variant string, exp int64) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte("media:" + photoID + ":" + variant + ":" + strconv.FormatInt(exp, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// URL returns a path valid for between one and two hours. Rounding the expiry to
// the hour keeps the URL stable across requests, so HTTP caches keep working.
func (s *Signer) URL(photoID, variant string, now time.Time) string {
	exp := (now.Unix()/3600 + 2) * 3600
	q := url.Values{}
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", s.sign(photoID, variant, exp))
	return "/media/photos/" + photoID + "/" + variant + "?" + q.Encode()
}

func (s *Signer) Verify(photoID, variant, expRaw, sig string, now time.Time) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || now.Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(s.sign(photoID, variant, exp)), []byte(sig))
}
