// Package storage persists binary objects (photos) outside the database.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrNotFound   = errors.New("object not found")
	ErrInvalidKey = errors.New("invalid storage key")
)

// Store is the abstraction used by features; LocalStore implements it on disk and an
// S3-compatible implementation can be added without touching feature code.
type Store interface {
	Put(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}

// Keys are generated server-side; this pattern rejects any path traversal attempt.
var keyPattern = regexp.MustCompile(`^[a-z0-9]+(/[a-z0-9-]+)*\.[a-z0-9]+$`)

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

func (s *LocalStore) Put(_ context.Context, key string, data []byte) error {
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
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o640); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}

func (s *LocalStore) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	target, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
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
