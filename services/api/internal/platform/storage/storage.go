// Package storage abstracts binary object storage (user photos). The default
// implementation writes to a local directory mounted as a Docker volume; an
// S3-compatible backend can implement the same interface later.
package storage

import (
	"context"
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

var ErrNotFound = errors.New("object not found")

// keyPattern is the only accepted key format. It rules out path traversal and
// arbitrary file names entirely.
var keyPattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

type Store interface {
	Put(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, time.Time, error)
	Delete(ctx context.Context, key string) error
}

// NewKey returns a random, unguessable object key.
func NewKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}

func ValidKey(key string) bool { return keyPattern.MatchString(key) }

type LocalStore struct {
	dir string
}

func NewLocalStore(dir string) (*LocalStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStore{dir: dir}, nil
}

func (s *LocalStore) path(key string) (string, error) {
	if !ValidKey(key) {
		return "", ErrNotFound
	}
	// Two-level fan-out keeps directories small.
	return filepath.Join(s.dir, key[:2], key), nil
}

func (s *LocalStore) Put(_ context.Context, key string, data []byte) error {
	p, err := s.path(key)
	if err != nil {
		return errors.New("invalid storage key")
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

func (s *LocalStore) Open(_ context.Context, key string) (io.ReadSeekCloser, time.Time, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, time.Time{}, err
	}
	f, err := os.Open(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, time.Time{}, ErrNotFound
		}
		return nil, time.Time{}, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, time.Time{}, err
	}
	return f, info.ModTime(), nil
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return nil
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// URLSigner issues short-lived signed media URLs. Photos are only reachable
// through URLs handed out by authorised API responses (discovery, matches,
// own profile), so blocking someone or deleting a photo stops access once the
// URL expires.
type URLSigner struct {
	secret []byte
	window time.Duration
	now    func() time.Time
}

func NewURLSigner(secret []byte, window time.Duration) *URLSigner {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("media-url-signing-v1"))
	return &URLSigner{secret: mac.Sum(nil), window: window, now: time.Now}
}

// Sign returns a relative URL. Expiry is aligned on window boundaries so the
// same photo keeps the same URL for a while, which lets clients cache it.
func (s *URLSigner) Sign(key string) string {
	now := s.now().Unix()
	window := int64(s.window.Seconds())
	exp := (now/window + 2) * window
	return "/media/" + key + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.signature(key, exp)
}

func (s *URLSigner) Verify(key, rawExp, sig string) bool {
	exp, err := strconv.ParseInt(rawExp, 10, 64)
	if err != nil || exp < s.now().Unix() {
		return false
	}
	expected := s.signature(key, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *URLSigner) signature(key string, exp int64) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(key))
	mac.Write([]byte{0})
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(mac.Sum(nil))[:40]
}
