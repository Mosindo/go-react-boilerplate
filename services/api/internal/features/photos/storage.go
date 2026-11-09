package photos

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Storage abstracts where photo bytes live (disk today, S3-compatible later).
type Storage interface {
	Save(key string, data []byte) error
	Read(key string) ([]byte, error)
	Delete(key string) error
}

var keyRe = regexp.MustCompile(`^[a-f0-9]{32}\.jpg$`)

func NewKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ".jpg", nil
}

// DiskStorage stores files in a private directory that is never served statically:
// every read goes through an authorised API endpoint.
type DiskStorage struct{ dir string }

func NewDiskStorage(dir string) (*DiskStorage, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &DiskStorage{dir: dir}, nil
}

func (d *DiskStorage) path(key string) (string, error) {
	if !keyRe.MatchString(key) {
		return "", errors.New("invalid storage key")
	}
	return filepath.Join(d.dir, key), nil
}

func (d *DiskStorage) Save(key string, data []byte) error {
	p, err := d.path(key)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func (d *DiskStorage) Read(key string) ([]byte, error) {
	p, err := d.path(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

func (d *DiskStorage) Delete(key string) error {
	p, err := d.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
