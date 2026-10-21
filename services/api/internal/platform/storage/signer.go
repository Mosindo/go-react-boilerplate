package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const urlWindow = time.Hour

// Signer builds and verifies expiring media URLs. Photos are private: a URL is only handed to
// users allowed to see the profile, and stops working after at most two windows.
type Signer struct{ key []byte }

func NewSigner(jwtSecret []byte) *Signer {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte("alma-media-v1"))
	return &Signer{key: mac.Sum(nil)}
}

func (s *Signer) sign(id, size string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	fmt.Fprintf(mac, "%s|%s|%d", id, size, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

// URL returns a relative path. The expiry is aligned on window boundaries so the URL stays stable
// (and cacheable by the client) for up to an hour.
func (s *Signer) URL(photoID, size string) string {
	exp := (time.Now().Unix()/int64(urlWindow.Seconds()) + 2) * int64(urlWindow.Seconds())
	q := url.Values{}
	q.Set("size", size)
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", s.sign(photoID, size, exp))
	return "/media/photos/" + photoID + "?" + q.Encode()
}

func (s *Signer) Verify(photoID, size, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	expected := s.sign(photoID, size, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}

// ThumbURL signs the thumbnail of an optional photo id; it returns "" when the user has no photo.
func (s *Signer) ThumbURL(photoID *string) string {
	if photoID == nil {
		return ""
	}
	return s.URL(*photoID, "thumb")
}
