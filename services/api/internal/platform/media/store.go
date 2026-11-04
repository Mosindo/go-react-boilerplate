package media

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var keyPattern = regexp.MustCompile(`^[0-9a-f]{32}\.jpg$`)

// Store persists photo files on local disk, outside any statically served path.
// Swap this type for an object-storage implementation to scale horizontally.
type Store struct {
	dir string
}

// NewStore creates the upload directory if needed.
func NewStore(dir string) (*Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Store{dir: abs}, nil
}

// Put writes data under a fresh random key and returns the key.
func (s *Store) Put(data []byte) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	key := hex.EncodeToString(raw) + ".jpg"
	if err := os.WriteFile(filepath.Join(s.dir, key), data, 0o640); err != nil {
		return "", err
	}
	return key, nil
}

// Path resolves a key to a file path. Keys that do not match the generated
// format are rejected, so path traversal is impossible.
func (s *Store) Path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", errors.New("invalid storage key")
	}
	return filepath.Join(s.dir, key), nil
}

// Delete removes a file; a missing file is not an error.
func (s *Store) Delete(key string) error {
	p, err := s.Path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
