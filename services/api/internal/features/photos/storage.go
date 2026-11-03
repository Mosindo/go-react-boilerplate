package photos

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Storage abstracts where photo bytes live (local disk today, S3 later).
type Storage interface {
	Save(key string, data []byte) error
	Open(key string) (*os.File, error)
	Delete(key string) error
}

// LocalStorage keeps files under a root directory. Keys are generated
// server-side (<userID>/<uuid>.jpg) and still validated against traversal.
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
	if key == "" || strings.Contains(key, "..") || filepath.IsAbs(key) {
		return "", errors.New("invalid storage key")
	}
	full := filepath.Join(s.root, filepath.FromSlash(key))
	if !strings.HasPrefix(full, s.root+string(os.PathSeparator)) {
		return "", errors.New("invalid storage key")
	}
	return full, nil
}

func (s *LocalStorage) Save(key string, data []byte) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func (s *LocalStorage) Open(key string) (*os.File, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (s *LocalStorage) Delete(key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// RemoveUserFiles deletes every file stored for a user (account erasure).
func (s *LocalStorage) RemoveUserFiles(_ context.Context, userID string) error {
	p, err := s.path(userID)
	if err != nil {
		return err
	}
	return os.RemoveAll(p)
}
