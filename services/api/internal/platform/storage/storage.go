// Package storage abstracts where binary files (photos) live. The local disk
// implementation is the default; implement Storage for S3/GCS in production.
package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrNotFound = errors.New("storage: object not found")

type Storage interface {
	Put(key string, data []byte) error
	Open(key string) (io.ReadSeekCloser, error)
	Delete(keys ...string) error
}

type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &Local{root: abs}, nil
}

// path resolves a key inside root and refuses anything that escapes it.
func (l *Local) path(key string) (string, error) {
	if key == "" || strings.Contains(key, "..") || filepath.IsAbs(key) {
		return "", errors.New("storage: invalid key")
	}
	full := filepath.Join(l.root, filepath.FromSlash(key))
	if !strings.HasPrefix(full, l.root+string(os.PathSeparator)) {
		return "", errors.New("storage: invalid key")
	}
	return full, nil
}

func (l *Local) Put(key string, data []byte) error {
	full, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return err
	}
	tmp := full + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, full)
}

func (l *Local) Open(key string) (io.ReadSeekCloser, error) {
	full, err := l.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func (l *Local) Delete(keys ...string) error {
	var firstErr error
	for _, key := range keys {
		full, err := l.path(key)
		if err == nil {
			err = os.Remove(full)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
