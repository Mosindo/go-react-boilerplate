// Package media signs short-lived URLs for private media so images can be loaded by
// <Image> components without exposing bearer tokens or making files public.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// Window controls URL stability: URLs stay identical inside a window (cache friendly)
// and remain valid between one and two windows.
const Window = 6 * time.Hour

type Signer struct {
	key []byte
	now func() time.Time
}

func NewSigner(secret []byte) *Signer {
	// Derive a dedicated key so media signatures can never be confused with JWTs.
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("media-url-signing-v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID + "|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// PhotoURL returns a relative signed URL for a photo.
func (s *Signer) PhotoURL(photoID string) string {
	windowSeconds := int64(Window.Seconds())
	exp := (s.now().Unix()/windowSeconds + 2) * windowSeconds
	return fmt.Sprintf("/media/photos/%s?exp=%d&sig=%s", photoID, exp, s.sign(photoID, exp))
}

// Verify checks signature and expiry for a photo URL.
func (s *Signer) Verify(photoID, rawExp, sig string) bool {
	exp, err := strconv.ParseInt(rawExp, 10, 64)
	if err != nil || exp < s.now().Unix() {
		return false
	}
	expected := s.sign(photoID, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}
