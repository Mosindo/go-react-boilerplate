// Package media issues short-lived signed URLs for private member photos.
// Photo files are never publicly listable: a URL is only handed out to members
// allowed to see the profile, and it expires.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

const (
	// URLs are valid for at least urlTTL and are bucketed so that the same
	// photo keeps a stable URL for a while (better client-side caching).
	urlTTL    = time.Hour
	urlBucket = time.Hour
)

type Variant string

const (
	VariantFull  Variant = "full"
	VariantThumb Variant = "thumb"
)

type Signer struct {
	secret []byte
	now    func() time.Time
}

// NewSigner derives the signing key from secret with a fixed label so the
// media key differs from the JWT key even when derived from JWT_SECRET.
func NewSigner(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("media-url-signing-v1"))
	return &Signer{secret: mac.Sum(nil), now: time.Now}
}

func (s *Signer) signature(photoID string, variant Variant, exp int64) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(photoID + "|" + string(variant) + "|" + strconv.FormatInt(exp, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// URL returns a relative URL for the photo variant. Clients prefix it with
// the API base URL.
func (s *Signer) URL(photoID string, variant Variant) string {
	now := s.now().UTC()
	exp := now.Truncate(urlBucket).Add(urlBucket + urlTTL).Unix()
	return "/media/photos/" + photoID + "?v=" + string(variant) + "&exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.signature(photoID, variant, exp)
}

// Verify checks a signature and its expiry.
func (s *Signer) Verify(photoID string, variant Variant, expRaw, sig string) bool {
	if variant != VariantFull && variant != VariantThumb {
		return false
	}
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().UTC().Unix() > exp {
		return false
	}
	expected := s.signature(photoID, variant, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}
