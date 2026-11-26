// Package media provides photo storage, image sanitisation and signed media URLs.
package media

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var ErrInvalidKey = errors.New("invalid storage key")

// Storage abstracts where photo bytes live. LocalStorage is the default; an object-storage
// implementation (S3, GCS...) can be added without touching features.
type Storage interface {
	Save(key string, data []byte) error
	Open(key string) (io.ReadSeekCloser, time.Time, error)
	Delete(key string) error
}

var keyPattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

// NewKey returns a random, unguessable storage key. Keys never contain user input.
func NewKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}

func ValidKey(key string) bool {
	return keyPattern.MatchString(key)
}

type LocalStorage struct {
	dir string
}

func NewLocalStorage(dir string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStorage{dir: dir}, nil
}

func (s *LocalStorage) path(key string) (string, error) {
	if !ValidKey(key) {
		return "", ErrInvalidKey
	}
	// Two-level fan-out keeps directories small.
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

func (s *LocalStorage) Open(key string) (io.ReadSeekCloser, time.Time, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, time.Time{}, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, time.Time{}, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, time.Time{}, err
	}
	return f, info.ModTime(), nil
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
