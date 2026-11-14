// Package storage abstracts binary object storage. The default
// implementation writes to the local filesystem; an S3-compatible backend
// can be added behind the same interface without touching features.
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

var ErrNotFound = errors.New("object not found")

var keyPattern = regexp.MustCompile(`^[a-z0-9]+(/[a-z0-9]+)*\.[a-z0-9]+$`)

type Storage interface {
	Put(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}

type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve upload dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Local{root: abs}, nil
}

// path validates the key (generated server-side, never user supplied) and
// maps it inside the storage root, preventing any path traversal.
func (l *Local) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	return filepath.Join(l.root, filepath.FromSlash(key)), nil
}

func (l *Local) Put(_ context.Context, key string, data []byte) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (l *Local) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	p, err := l.path(key)
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

func (l *Local) Delete(_ context.Context, key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
