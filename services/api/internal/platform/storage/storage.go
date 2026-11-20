// Package storage persists uploaded binary objects. The local disk backend is
// the default; another backend (S3, GCS...) only needs to implement Store.
package storage

import (
	"context"
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
	ErrNotFound   = errors.New("storage object not found")
	ErrInvalidKey = errors.New("invalid storage key")
)

type Store interface {
	Put(ctx context.Context, key string, r io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// keyPattern only allows server-generated keys, which rules out path traversal.
var keyPattern = regexp.MustCompile(`^[a-z0-9]{2}/[a-f0-9]{32}\.(jpg)$`)

// NewKey returns a random, unguessable key such as "ab/ab12....jpg".
func NewKey(extension string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	name := hex.EncodeToString(buf)
	return name[:2] + "/" + name + "." + extension, nil
}

type LocalStore struct {
	root string
}

func NewLocalStore(root string) (*LocalStore, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve upload dir: %w", err)
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

func (s *LocalStore) Put(_ context.Context, key string, r io.Reader) error {
	target, err := s.path(key)
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
	return os.Rename(tmpName, target)
}

func (s *LocalStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	target, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
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
