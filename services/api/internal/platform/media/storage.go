// Package media stores member photos on disk, re-encodes uploads into safe
// JPEGs and issues short-lived signed URLs so photos are never publicly
// enumerable.
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
)

var (
	ErrInvalidKey = errors.New("invalid storage key")
	ErrNotFound   = errors.New("media not found")
)

var storageKeyPattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

// Storage abstracts the blob store so a cloud bucket can replace the local
// disk implementation without touching features.
type Storage interface {
	Save(key string, r io.Reader) error
	Open(key string) (*os.File, error)
	Delete(key string) error
	Path(key string) (string, error)
}

// DiskStorage stores blobs under root/<first two chars>/<key>.
type DiskStorage struct {
	root string
}

func NewDiskStorage(root string) (*DiskStorage, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve media dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create media dir: %w", err)
	}
	return &DiskStorage{root: abs}, nil
}

// NewKey returns a random, unguessable storage key.
func NewKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}

func (s *DiskStorage) Path(key string) (string, error) {
	if !storageKeyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(s.root, key[:2], key), nil
}

func (s *DiskStorage) Save(key string, r io.Reader) error {
	target, err := s.Path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, r); err != nil {
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
	return os.Rename(tmpName, target)
}

func (s *DiskStorage) Open(key string) (*os.File, error) {
	target, err := s.Path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *DiskStorage) Delete(key string) error {
	target, err := s.Path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
