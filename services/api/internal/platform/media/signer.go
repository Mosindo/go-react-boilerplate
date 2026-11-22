package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

// Signer issues capability URLs for photos. URLs are only handed to viewers
// allowed to see a profile; they expire, and the expiry is bucketed so the
// same URL is reused for a while (good for client-side image caching).
type Signer struct {
	key    []byte
	window time.Duration
}

func NewSigner(secret []byte, window time.Duration) *Signer {
	// Domain separation: never reuse the raw JWT key for another purpose.
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("media-url-signing-v1"))
	return &Signer{key: mac.Sum(nil), window: window}
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID))
	mac.Write([]byte{0})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Expiry returns an expiry between one and two windows after now.
func (s *Signer) Expiry(now time.Time) int64 {
	w := int64(s.window.Seconds())
	return (now.Unix()/w + 2) * w
}

// PhotoURL returns a relative URL that the client prefixes with its API base.
func (s *Signer) PhotoURL(photoID string, now time.Time) string {
	exp := s.Expiry(now)
	q := url.Values{}
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", s.sign(photoID, exp))
	return "/media/photos/" + photoID + "?" + q.Encode()
}

// Verify checks the signature and expiry in constant time.
func (s *Signer) Verify(photoID, rawExp, sig string, now time.Time) (time.Time, bool) {
	exp, err := strconv.ParseInt(rawExp, 10, 64)
	if err != nil || exp < now.Unix() || exp > now.Add(3*s.window).Unix() {
		return time.Time{}, false
	}
	expected := s.sign(photoID, exp)
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return time.Time{}, false
	}
	return time.Unix(exp, 0), true
}
