// Package storage abstracts where uploaded files live. The local-disk
// implementation is the default; swap in an S3-style store behind Store.
package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var ErrNotFound = errors.New("stored file not found")

var keyPattern = regexp.MustCompile(`^[a-f0-9-]{36}\.jpg$`)

type Store interface {
	Save(key string, data []byte) error
	Read(key string) ([]byte, error)
	Delete(key string) error
}

type LocalStore struct {
	dir string
}

func NewLocalStore(dir string) (*LocalStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStore{dir: dir}, nil
}

// path refuses anything that is not a server generated key, which rules out traversal.
func (s *LocalStore) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", fmt.Errorf("invalid storage key")
	}
	return filepath.Join(s.dir, key), nil
}

func (s *LocalStore) Save(key string, data []byte) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o640)
}

func (s *LocalStore) Read(key string) ([]byte, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return data, err
}

func (s *LocalStore) Delete(key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
