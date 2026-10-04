// Package storage abstracts binary file persistence (photos). The local filesystem
// implementation is the default; an S3-style implementation can satisfy Store later.
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("storage: object not found")

type Store interface {
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

// path maps a key to a file below root, refusing anything that could escape it.
func (l *Local) path(key string) (string, error) {
	if key == "" || strings.ContainsAny(key, "\\\x00") || strings.HasPrefix(key, "/") {
		return "", fmt.Errorf("storage: invalid key")
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("storage: invalid key")
		}
	}
	full := filepath.Join(l.root, filepath.FromSlash(key))
	if !strings.HasPrefix(full, l.root+string(filepath.Separator)) {
		return "", fmt.Errorf("storage: invalid key")
	}
	return full, nil
}

func (l *Local) Put(_ context.Context, key string, data []byte) error {
	full, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return fmt.Errorf("storage mkdir: %w", err)
	}
	tmp := filepath.Join(filepath.Dir(full), ".tmp-"+uuid.NewString())
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return fmt.Errorf("storage write: %w", err)
	}
	if err := os.Rename(tmp, full); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("storage rename: %w", err)
	}
	return nil
}

func (l *Local) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	full, err := l.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	full, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Memory is an in-memory Store for tests.
type Memory struct {
	Files map[string][]byte
}

func NewMemory() *Memory { return &Memory{Files: map[string][]byte{}} }

func (m *Memory) Put(_ context.Context, key string, data []byte) error {
	m.Files[key] = append([]byte(nil), data...)
	return nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func (m *Memory) Open(_ context.Context, key string) (io.ReadSeekCloser, error) {
	data, ok := m.Files[key]
	if !ok {
		return nil, ErrNotFound
	}
	return nopCloser{bytes.NewReader(data)}, nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	delete(m.Files, key)
	return nil
}
