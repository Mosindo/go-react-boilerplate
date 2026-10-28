// Package storage persists binary blobs (photos) outside the web root, addressed by opaque keys.
package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var keyPattern = regexp.MustCompile(`^[a-f0-9]{2}/[a-f0-9-]{36}\.jpg$`)

var ErrInvalidKey = errors.New("invalid storage key")

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

// Key builds the on-disk key for an id; keys are never derived from user-provided names.
func Key(id string) string {
	return id[:2] + "/" + id + ".jpg"
}

func (l *Local) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", ErrInvalidKey
	}
	return filepath.Join(l.root, filepath.FromSlash(key)), nil
}

func (l *Local) Put(key string, data []byte) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o640)
}

func (l *Local) Open(key string) (io.ReadSeekCloser, os.FileInfo, error) {
	p, err := l.path(key)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	return f, info, nil
}

// Remove deletes a blob; a missing file is not an error.
func (l *Local) Remove(key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
