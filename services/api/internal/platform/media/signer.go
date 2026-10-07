// Package media produces and verifies short-lived signed URLs for photos, so images
// can be loaded by plain <Image> components (and cached) while staying private.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const (
	bucket = 6 * time.Hour
	ttl    = 12 * time.Hour
)

type Signer struct {
	key []byte
}

func NewSigner(secret []byte) *Signer {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("media-url-v1"))
	return &Signer{key: mac.Sum(nil)}
}

func (s *Signer) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(photoID + "." + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// URL returns a relative URL valid for at least 12h. The expiry is bucketed so the
// URL is stable for 6h, which keeps client image caches effective.
func (s *Signer) URL(photoID string) string {
	exp := time.Now().Truncate(bucket).Add(bucket + ttl).Unix()
	return fmt.Sprintf("/media/%s?exp=%d&sig=%s", photoID, exp, s.sign(photoID, exp))
}

// Verify checks signature and expiry.
func (s *Signer) Verify(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	want := s.sign(photoID, exp)
	return hmac.Equal([]byte(want), []byte(sig))
}

// Photo is the API representation of a stored picture.
type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}
