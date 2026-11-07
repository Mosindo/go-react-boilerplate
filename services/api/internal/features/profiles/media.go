package profiles

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

const mediaURLValidity = time.Hour

// MediaSigner issues short-lived, HMAC-signed photo URLs. Authorization is decided when a URL is
// generated (only for viewers allowed to see the profile); the signature then protects the bytes.
// Expiry is bucketed per hour so a URL stays stable (image caches keep working) within a bucket.
type MediaSigner struct {
	key []byte
	now func() time.Time
}

func NewMediaSigner(jwtSecret []byte) *MediaSigner {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte("media-url-v1"))
	return &MediaSigner{key: mac.Sum(nil), now: time.Now}
}

func (m *MediaSigner) sign(photoID string, exp int64) string {
	mac := hmac.New(sha256.New, m.key)
	mac.Write([]byte(photoID + "|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (m *MediaSigner) URL(photoID string) string {
	bucket := int64(mediaURLValidity.Seconds())
	exp := (m.now().Unix()/bucket + 2) * bucket
	return "/media/" + photoID + "?e=" + strconv.FormatInt(exp, 10) + "&s=" + m.sign(photoID, exp)
}

func (m *MediaSigner) Verify(photoID, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || exp < m.now().Unix() {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(m.sign(photoID, exp)))
}
