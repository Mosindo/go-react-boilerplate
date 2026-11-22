// Package media stores user images and issues short-lived signed URLs for them.
package media

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrInvalidKey = errors.New("invalid storage key")
	ErrNotFound   = errors.New("media not found")
)

// Storage abstracts where image bytes live. LocalStorage is the default; an
// object-storage implementation (S3, GCS, R2...) can be swapped in without
// touching feature code.
type Storage interface {
	Save(key string, data []byte) error
	Open(key string) (io.ReadSeekCloser, error)
	Delete(key string) error
}

// Keys are generated server-side; this pattern rejects anything that could
// escape the storage directory.
var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}\.(jpg)$`)

type LocalStorage struct {
	dir string
}

func NewLocalStorage(dir string) (*LocalStorage, error) {
	if dir == "" {
		return nil, errors.New("upload directory is required")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}
	return &LocalStorage{dir: dir}, nil
}

func (s *LocalStorage) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(s.dir, key), nil
}

func (s *LocalStorage) Save(key string, data []byte) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, p)
}

func (s *LocalStorage) Open(key string) (io.ReadSeekCloser, error) {
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
