// Package storage abstracts where uploaded binary files live. The local-disk
// implementation is the default; an S3-compatible store can implement Store.
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

var ErrNotFound = errors.New("storage: object not found")

// keyPattern is the only shape of key the application generates; validating it
// makes path traversal impossible by construction.
var keyPattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

type Store interface {
	Put(ctx context.Context, key string, data io.Reader) error
	Open(ctx context.Context, key string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, key string) error
}

type Local struct {
	dir string
}

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &Local{dir: abs}, nil
}

func (l *Local) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return filepath.Join(l.dir, key), nil
}

func (l *Local) Put(_ context.Context, key string, data io.Reader) error {
	target, err := l.path(key)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(l.dir, ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, data); err != nil {
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
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

func (l *Local) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	target, err := l.path(key)
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

func (l *Local) Delete(_ context.Context, key string) error {
	target, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
