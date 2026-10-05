package photos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var keyPattern = regexp.MustCompile(`^[0-9a-f]{32}\.jpg$`)

var ErrStorageNotFound = errors.New("stored file not found")

// Storage keeps processed photo bytes. Swap the implementation (S3, GCS) without touching the feature.
type Storage interface {
	Put(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}

// LocalStorage stores files in a private directory. Files are never served by path: only through the
// signed-URL handler, and keys are random so they cannot be guessed or traversed.
type LocalStorage struct{ root string }

func NewLocalStorage(root string) (*LocalStorage, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStorage{root: abs}, nil
}

func (s *LocalStorage) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	return filepath.Join(s.root, key), nil
}

func (s *LocalStorage) Put(_ context.Context, key string, data []byte) error {
	dst, err := s.path(key)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.root, ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrStorageNotFound
	}
	return f, err
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
