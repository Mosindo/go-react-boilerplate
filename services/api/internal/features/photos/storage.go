package photos

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Storage keeps processed images on local disk. Keys are server-generated
// (never client-controlled) and every path is checked to stay under root.
type Storage struct {
	root string
}

func NewStorage(root string) (*Storage, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &Storage{root: abs}, nil
}

func (s *Storage) path(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", errors.New("invalid storage key")
	}
	return filepath.Join(s.root, clean), nil
}

func variantKey(key, variant string) string {
	if variant == "thumb" {
		return strings.TrimSuffix(key, ".jpg") + "_thumb.jpg"
	}
	return key
}

// Save writes both variants of an image.
func (s *Storage) Save(key string, full, thumb []byte) error {
	for variant, data := range map[string][]byte{"full": full, "thumb": thumb} {
		p, err := s.path(variantKey(key, variant))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(p, data, 0o640); err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) Open(key, variant string) (*os.File, error) {
	p, err := s.path(variantKey(key, variant))
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Remove deletes both variants of each key. Missing files are ignored.
func (s *Storage) Remove(keys ...string) {
	for _, key := range keys {
		for _, variant := range []string{"full", "thumb"} {
			if p, err := s.path(variantKey(key, variant)); err == nil {
				_ = os.Remove(p)
			}
		}
	}
}
