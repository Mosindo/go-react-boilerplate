package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

// Signer issues short-lived, HMAC-signed media URLs. Photos are only reachable through
// URLs handed out by endpoints that already enforced visibility rules (blocks, moderation).
// Expiry is bucketed so a URL stays stable for a while, which keeps client image caches useful.
type Signer struct {
	key    []byte
	bucket time.Duration
	now    func() time.Time
}

func NewSigner(appSecret []byte) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("media-url-signing-v1"))
	return &Signer{key: mac.Sum(nil), bucket: time.Hour, now: time.Now}
}

// URL returns a relative URL ("/media/<key>?exp=..&sig=..") valid for one to two hours.
func (s *Signer) URL(key string) string {
	exp := s.now().Truncate(s.bucket).Add(2 * s.bucket).Unix()
	expStr := strconv.FormatInt(exp, 10)
	return "/media/" + key + "?exp=" + expStr + "&sig=" + s.sign(key, expStr)
}

func (s *Signer) Verify(key, exp, sig string) bool {
	expUnix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || s.now().Unix() > expUnix {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(s.sign(key, exp)))
}

func (s *Signer) sign(key, exp string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(key + "|" + exp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
