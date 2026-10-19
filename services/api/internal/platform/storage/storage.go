package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrNotFound   = errors.New("storage object not found")
	ErrInvalidKey = errors.New("invalid storage key")
)

var keyPattern = regexp.MustCompile(`^[a-f0-9-]{36}\.jpg$`)

// Store persists processed photo bytes. Keys are server-generated and strictly
// validated so a key can never escape the storage root.
type Store interface {
	Put(key string, data []byte) error
	Open(key string) (io.ReadSeekCloser, error)
	Delete(key string) error
}

type LocalStore struct {
	root string
}

func NewLocalStore(root string) (*LocalStore, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStore{root: root}, nil
}

func (s *LocalStore) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(s.root, key[:2], key), nil
}

func (s *LocalStore) Put(key string, data []byte) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, p)
}

func (s *LocalStore) Open(key string) (io.ReadSeekCloser, error) {
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
