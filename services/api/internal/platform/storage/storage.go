// Package storage persists binary files on local disk under a root directory.
// Keys are opaque server-generated names; callers never pass user input as a path.
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var keyPattern = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

type Disk struct{ root string }

func NewDisk(root string) (*Disk, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Disk{root: abs}, nil
}

// Put stores data under a fresh random key and returns the key.
func (d *Disk) Put(data []byte) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	key := hex.EncodeToString(buf) + ".jpg"
	if err := os.WriteFile(filepath.Join(d.root, key), data, 0o640); err != nil {
		return "", err
	}
	return key, nil
}

func (d *Disk) Read(key string) ([]byte, error) {
	if !keyPattern.MatchString(key) {
		return nil, errors.New("invalid storage key")
	}
	return os.ReadFile(filepath.Join(d.root, key))
}

func (d *Disk) Delete(key string) {
	if keyPattern.MatchString(key) {
		_ = os.Remove(filepath.Join(d.root, key))
	}
}
