// Package storage persists binary objects (profile photos). The Local
// implementation writes to disk; another implementation (S3, GCS...) only
// needs to satisfy Storage.
package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrNotFound   = errors.New("object not found")
	ErrInvalidKey = errors.New("invalid storage key")
)

// Keys are generated server-side: 32 hex chars + extension. Anything else is
// rejected, which rules out path traversal.
var keyPattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

func ValidKey(key string) bool { return keyPattern.MatchString(key) }

type Storage interface {
	Put(key string, data []byte) error
	Open(key string) (*os.File, error)
	Delete(key string) error
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

func (l *Local) path(key string) (string, error) {
	if !ValidKey(key) {
		return "", ErrInvalidKey
	}
	// Shard by prefix to keep directories small.
	return filepath.Join(l.root, key[:2], key), nil
}

func (l *Local) Put(key string, data []byte) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".upload-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func (l *Local) Open(key string) (*os.File, error) {
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

func (l *Local) Delete(key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
