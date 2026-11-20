// Package mediaurl builds and verifies signed, expiring URLs for private media.
// Photos are never publicly listable: a URL is only handed out by an endpoint
// that already checked the viewer may see the profile.
package mediaurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

const (
	// URLs are bucketed so a given photo keeps the same URL for a while,
	// which lets clients cache images efficiently.
	bucket  = 6 * time.Hour
	minLife = 6 * time.Hour
)

type Signer struct {
	key []byte
	now func() time.Time
}

// NewSigner derives a dedicated signing key from the API secret so that a
// media signature can never be confused with a JWT signature.
func NewSigner(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("lueur/media-url/v1"))
	return &Signer{key: mac.Sum(nil), now: time.Now}
}

func (s *Signer) signature(photoID string, expires int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID + "." + strconv.FormatInt(expires, 10)))
	return hex.EncodeToString(mac.Sum(nil)[:20])
}

// PhotoURL returns a relative URL such as /media/photos/<id>?exp=..&sig=..
func (s *Signer) PhotoURL(photoID string) string {
	now := s.now().UTC()
	expires := now.Truncate(bucket).Add(bucket + minLife).Unix()
	query := url.Values{}
	query.Set("exp", strconv.FormatInt(expires, 10))
	query.Set("sig", s.signature(photoID, expires))
	return "/media/photos/" + photoID + "?" + query.Encode()
}

// Verify checks the signature and expiry of a media URL.
func (s *Signer) Verify(photoID, rawExpires, signature string) bool {
	expires, err := strconv.ParseInt(rawExpires, 10, 64)
	if err != nil || s.now().Unix() > expires {
		return false
	}
	expected := s.signature(photoID, expires)
	return hmac.Equal([]byte(expected), []byte(signature))
}
