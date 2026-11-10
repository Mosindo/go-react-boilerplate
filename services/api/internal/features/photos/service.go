package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"strings"

	"example.com/api/internal/platform/storage"
)

var (
	ErrNotFound     = errors.New("photo not found")
	ErrLimitReached = errors.New("you can upload at most 6 photos")
	ErrInvalidOrder = errors.New("order must list each of your photos exactly once")
	ErrDuplicateID  = errors.New("duplicate photo id")
)

type Service struct {
	repo  Repository
	store storage.Store
}

func NewService(repo Repository, store storage.Store) *Service {
	return &Service{repo: repo, store: store}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (Photo, error) {
	processed, err := Process(raw)
	if err != nil {
		return Photo{}, err
	}
	key, err := newKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.store.Put(ctx, key, processed.Data); err != nil {
		return Photo{}, err
	}
	photo, err := s.repo.Insert(ctx, userID, key, processed.Width, processed.Height, len(processed.Data))
	if err != nil {
		s.deleteFile(ctx, key)
		if errors.Is(err, ErrRepoLimit) {
			return Photo{}, ErrLimitReached
		}
		return Photo{}, err
	}
	return photo, nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) error {
	processed, err := Process(raw)
	if err != nil {
		return err
	}
	key, err := newKey()
	if err != nil {
		return err
	}
	if err := s.store.Put(ctx, key, processed.Data); err != nil {
		return err
	}
	old, err := s.repo.Replace(ctx, userID, photoID, key, processed.Width, processed.Height, len(processed.Data))
	if err != nil {
		s.deleteFile(ctx, key)
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.deleteFile(ctx, old)
	return nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.deleteFile(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(id)
		if seen[id] {
			return ErrDuplicateID
		}
		seen[id] = true
		normalized = append(normalized, id)
	}
	err := s.repo.Reorder(ctx, userID, normalized)
	if errors.Is(err, ErrRepoMismatch) {
		return ErrInvalidOrder
	}
	return err
}

// DeleteFiles removes stored files (used after account deletion).
func (s *Service) DeleteFiles(ctx context.Context, keys []string) {
	for _, key := range keys {
		s.deleteFile(ctx, key)
	}
}

func (s *Service) deleteFile(ctx context.Context, key string) {
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"photo_file_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}

func newKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}
