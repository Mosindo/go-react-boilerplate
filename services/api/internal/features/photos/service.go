package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"

	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo  Repository
	store storage.Storage
}

func NewService(repo Repository, store storage.Storage) *Service {
	return &Service{repo: repo, store: store}
}

func newKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ".jpg", nil
}

func (s *Service) save(raw []byte) (string, Processed, error) {
	img, err := Process(raw)
	if err != nil {
		return "", Processed{}, err
	}
	key, err := newKey()
	if err != nil {
		return "", Processed{}, err
	}
	if err := s.store.Put(key, img.Data); err != nil {
		return "", Processed{}, err
	}
	return key, img, nil
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) error {
	key, img, err := s.save(raw)
	if err != nil {
		return err
	}
	if _, err := s.repo.Insert(ctx, userID, key, img.Width, img.Height, len(img.Data)); err != nil {
		s.DeleteObjects([]string{key})
		return err
	}
	return nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) error {
	key, img, err := s.save(raw)
	if err != nil {
		return err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.DeleteObjects([]string{key})
		return err
	}
	s.DeleteObjects([]string{oldKey})
	return nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.DeleteObjects([]string{key})
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	seen := map[string]bool{}
	for _, id := range photoIDs {
		if seen[id] {
			return ErrInvalidOrder
		}
		seen[id] = true
	}
	return s.repo.Reorder(ctx, userID, photoIDs)
}

// StorageKeysForUser and DeleteObjects implement auth.AccountDataCleaner.
func (s *Service) StorageKeysForUser(ctx context.Context, userID string) ([]string, error) {
	return s.repo.StorageKeysForUser(ctx, userID)
}

func (s *Service) DeleteObjects(keys []string) {
	for _, key := range keys {
		if err := s.store.Delete(key); err != nil {
			log.Printf(`{"event":"photo_delete_failed","key":%q,"error":%q}`, key, err.Error())
		}
	}
}

// Exists is used by the media endpoint so files of deleted photos (or deleted
// accounts) stop being served immediately, even with a still-valid signature.
func (s *Service) Exists(ctx context.Context, key string) (bool, error) {
	return s.repo.ExistsByKey(ctx, key)
}

func (s *Service) Store() storage.Storage { return s.store }
