package photos

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

var keyPattern = regexp.MustCompile(`^[0-9a-f]{32}\.jpg$`)

// DiskStorage stocke les photos sur le disque local, hors de toute racine servie
// statiquement. Les fichiers ne sont lisibles que via l'API (controle d'acces).
type DiskStorage struct{ dir string }

func NewDiskStorage(dir string) (*DiskStorage, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &DiskStorage{dir: dir}, nil
}

func newKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ".jpg", nil
}

func (s *DiskStorage) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", fmt.Errorf("invalid storage key")
	}
	return filepath.Join(s.dir, key), nil
}

func (s *DiskStorage) Save(key string, data []byte) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o640)
}

func (s *DiskStorage) Open(key string) (*os.File, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Remove supprime des fichiers (best effort, les erreurs sont journalisees).
func (s *DiskStorage) Remove(keys []string) {
	for _, k := range keys {
		p, err := s.path(k)
		if err != nil {
			continue
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			log.Printf(`{"event":"photo_remove_failed","error":%q}`, err.Error())
		}
	}
}
