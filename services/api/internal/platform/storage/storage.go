// Package storage persists binary blobs (profile photos) outside the database.
package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var ErrNotFound = errors.New("storage: object not found")

// keyPattern only allows server-generated keys, which rules out path traversal.
var keyPattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

type Store interface {
	Save(key string, data []byte) error
	Open(key string) (io.ReadCloser, error)
	Delete(key string) error
}

// LocalStore keeps objects in a private directory. Files are never served
// statically: the API streams them after an authorization check.
type LocalStore struct{ dir string }

func NewLocalStore(dir string) (*LocalStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStore{dir: dir}, nil
}

func (s *LocalStore) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", fmt.Errorf("storage: invalid key")
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

func (s *LocalStore) Open(key string) (io.ReadCloser, error) {
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
