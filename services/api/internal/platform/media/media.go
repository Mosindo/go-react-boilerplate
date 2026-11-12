// Package media stores user-uploaded images on disk and issues short-lived,
// HMAC-signed URLs so photos are never publicly enumerable.
package media

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

var (
	ErrInvalidKey = errors.New("invalid media key")
	ErrNotFound   = errors.New("media not found")
)

var keyPattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

// ValidKey reports whether key has the exact shape produced by NewKey. This
// is what prevents path traversal when serving files.
func ValidKey(key string) bool {
	return keyPattern.MatchString(key)
}

// NewKey returns a random, unguessable storage key.
func NewKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}

// Storage abstracts where image bytes live (local disk today, object storage later).
type Storage interface {
	Save(key string, data []byte) error
	Open(key string) (*os.File, error)
	Delete(key string) error
}

type LocalStorage struct {
	dir string
}

func NewLocalStorage(dir string) (*LocalStorage, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStorage{dir: abs}, nil
}

func (s *LocalStorage) path(key string) (string, error) {
	if !ValidKey(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(s.dir, key[:2], key), nil
}

func (s *LocalStorage) Save(key string, data []byte) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (s *LocalStorage) Open(key string) (*os.File, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *LocalStorage) Delete(key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Signer issues and verifies expiring media URLs.
type Signer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewSigner derives a dedicated signing key from the app secret so media
// signatures can never be confused with JWT signatures.
func NewSigner(appSecret []byte, ttl time.Duration) *Signer {
	mac := hmac.New(sha256.New, appSecret)
	_, _ = io.WriteString(mac, "media-url-signing-v1")
	return &Signer{secret: mac.Sum(nil), ttl: ttl, now: time.Now}
}

func (s *Signer) signature(key string, exp int64) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = io.WriteString(mac, key+"|"+strconv.FormatInt(exp, 10))
	return hex.EncodeToString(mac.Sum(nil))
}

// URL returns a relative, signed path for key. Expiry is aligned to the TTL
// boundary so the same photo keeps a stable URL (and client cache hit) for a while.
func (s *Signer) URL(key string) string {
	ttlSeconds := int64(s.ttl / time.Second)
	now := s.now().Unix()
	exp := (now/ttlSeconds + 2) * ttlSeconds
	return "/media/" + key + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.signature(key, exp)
}

func (s *Signer) Verify(key, expRaw, sig string) bool {
	if !ValidKey(key) {
		return false
	}
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || exp < s.now().Unix() {
		return false
	}
	expected := s.signature(key, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}
