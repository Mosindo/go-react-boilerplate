// Package storage abstracts where uploaded files live. The local-disk
// implementation is private to the server: files are only ever served through
// authorised API endpoints, never as static assets.
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

var ErrNotFound = errors.New("storage: not found")

type Store interface {
	Put(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9/_-]{0,120}\.[a-z0-9]{2,5}$`)

type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Local{root: abs}, nil
}

func (l *Local) path(key string) (string, error) {
	if !keyPattern.MatchString(key) || filepath.Clean(key) != key {
		return "", fmt.Errorf("storage: invalid key %q", key)
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
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
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
