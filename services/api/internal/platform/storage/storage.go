// Package storage persists binary objects (photos) outside the database.
// The local-disk implementation is the default; any object store can be
// plugged in by implementing Storage.
package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrNotFound   = errors.New("object not found")
	ErrInvalidKey = errors.New("invalid storage key")
	keyPattern    = regexp.MustCompile(`^[a-f0-9]{32,64}\.(jpg)$`)
)

type Storage interface {
	Save(ctx context.Context, key string, data []byte) error
	Path(key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type Local struct {
	dir string
}

func NewLocal(dir string) (*Local, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve upload dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Local{dir: abs}, nil
}

// Path returns the absolute file path for key. Keys are server-generated and
// validated against a strict pattern, which rules out path traversal.
func (l *Local) Path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(l.dir, key), nil
}

// Save writes data atomically (temp file + rename).
func (l *Local) Save(_ context.Context, key string, data []byte) error {
	path, err := l.Path(key)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(l.dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename upload: %w", err)
	}
	return nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	path, err := l.Path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
