// Package storage persists binary objects (photos) and signs short-lived
// URLs to read them. The local-disk backend is enough for a single node;
// an object-store backend only needs to implement Store.
package storage

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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
	ErrInvalidKey = errors.New("invalid storage key")
	ErrNotFound   = errors.New("object not found")
)

// Store is the minimal object storage contract used by features.
type Store interface {
	Put(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, time.Time, error)
	Delete(ctx context.Context, key string) error
}

// keys are always generated server-side; the pattern forbids traversal.
var keyPattern = regexp.MustCompile(`^[a-z]+/[0-9a-f-]{36}/[0-9a-f]{32}\.jpg$`)

// NewPhotoKey returns a random, unguessable key for a user's photo.
func NewPhotoKey(userID string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	key := "photos/" + userID + "/" + hex.EncodeToString(buf) + ".jpg"
	if !keyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return key, nil
}

// LocalStore writes objects below a root directory.
type LocalStore struct {
	root string
}

func NewLocalStore(root string) (*LocalStore, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStore{root: abs}, nil
}

func (s *LocalStore) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}

func (s *LocalStore) Put(_ context.Context, key string, data []byte) error {
	target, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

func (s *LocalStore) Open(_ context.Context, key string) (io.ReadSeekCloser, time.Time, error) {
	target, err := s.path(key)
	if err != nil {
		return nil, time.Time{}, err
	}
	f, err := os.Open(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, time.Time{}, ErrNotFound
		}
		return nil, time.Time{}, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, time.Time{}, err
	}
	return f, info.ModTime(), nil
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	target, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// URLSigner issues and verifies expiring HMAC signatures for media paths.
// Expiry is bucketed so a given photo keeps the same URL for a while, which
// lets clients cache images instead of re-downloading them on every fetch.
type URLSigner struct {
	key    []byte
	bucket time.Duration
	now    func() time.Time
}

// NewURLSigner derives a dedicated signing key from the application secret.
func NewURLSigner(appSecret []byte) *URLSigner {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("lueur/media-url/v1"))
	return &URLSigner{key: mac.Sum(nil), bucket: 6 * time.Hour, now: time.Now}
}

func (s *URLSigner) signature(id string, exp int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(id + "|" + strconv.FormatInt(exp, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// PhotoURL returns a relative, signed URL valid for between 6 and 12 hours.
func (s *URLSigner) PhotoURL(photoID string) string {
	now := s.now().Unix()
	b := int64(s.bucket.Seconds())
	exp := (now/b + 2) * b
	return "/media/photos/" + photoID + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.signature(photoID, exp)
}

// Verify checks a signature and its expiry in constant time.
func (s *URLSigner) Verify(id, expRaw, sig string) bool {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || exp < s.now().Unix() {
		return false
	}
	expected := s.signature(id, exp)
	return hmac.Equal([]byte(expected), []byte(sig))
}
