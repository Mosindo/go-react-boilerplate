// Package storage abstracts binary object persistence. The default
// implementation writes to a private directory on local disk; an object store
// (S3, GCS...) can be plugged in by implementing Store.
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
	"strings"
)

var ErrNotFound = errors.New("storage object not found")

type Store interface {
	Put(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}

// NewKey returns a random, unguessable object key with the given extension.
func NewKey(prefix, ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + "/" + hex.EncodeToString(buf) + ext, nil
}

type LocalStore struct {
	root string
}

func NewLocalStore(root string) (*LocalStore, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStore{root: abs}, nil
}

// resolve maps a key to a path and refuses anything escaping the root.
func (s *LocalStore) resolve(key string) (string, error) {
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") || strings.Contains(key, `\`) {
		return "", fmt.Errorf("invalid storage key")
	}
	path := filepath.Join(s.root, filepath.FromSlash(key))
	if !strings.HasPrefix(path, s.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid storage key")
	}
	return path, nil
}

func (s *LocalStore) Put(_ context.Context, key string, data []byte) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *LocalStore) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	path, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
