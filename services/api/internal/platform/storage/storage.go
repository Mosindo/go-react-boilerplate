// Package storage abstracts binary object storage. The local disk
// implementation is used by default; an S3-compatible backend can implement
// the same interface without touching feature code.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrNotFound = errors.New("object not found")

// Store persists opaque objects addressed by a relative key.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
}

// Local stores objects under a root directory on disk.
type Local struct {
	root string
}

// NewLocal creates the root directory if needed.
func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &Local{root: abs}, nil
}

// resolve maps a key to a path and rejects anything escaping the root.
func (l *Local) resolve(key string) (string, error) {
	cleaned := filepath.Clean("/" + strings.TrimSpace(key))
	if cleaned == "/" || strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	full := filepath.Join(l.root, cleaned)
	if !strings.HasPrefix(full, l.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	return full, nil
}

func (l *Local) Put(_ context.Context, key string, r io.Reader) error {
	path, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := l.resolve(key)
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

func (l *Local) Delete(_ context.Context, key string) error {
	path, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (l *Local) DeletePrefix(_ context.Context, prefix string) error {
	path, err := l.resolve(prefix)
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}
