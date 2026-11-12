package photos

import (
	"context"
	"log"

	"example.com/api/internal/platform/media"
)

type Service struct {
	repo    Repository
	storage media.Storage
	signer  *media.Signer
}

func NewService(repo Repository, storage media.Storage, signer *media.Signer) *Service {
	return &Service{repo: repo, storage: storage, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, Photo{ID: p.ID, URL: s.signer.URL(p.StorageKey), Width: p.Width, Height: p.Height, Position: p.Position})
	}
	return out, nil
}

// store sanitizes raw bytes and writes the result under a fresh random key.
func (s *Service) store(raw []byte) (newPhoto, error) {
	img, err := media.Process(raw)
	if err != nil {
		return newPhoto{}, err
	}
	key, err := media.NewKey()
	if err != nil {
		return newPhoto{}, err
	}
	if err := s.storage.Save(key, img.Data); err != nil {
		return newPhoto{}, err
	}
	return newPhoto{StorageKey: key, SizeBytes: len(img.Data), Width: img.Width, Height: img.Height}, nil
}

func (s *Service) discard(key string) {
	if err := s.storage.Delete(key); err != nil {
		log.Printf(`{"event":"photo_storage_cleanup_failed","error":%q}`, err.Error())
	}
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) ([]Photo, error) {
	p, err := s.store(raw)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Insert(ctx, userID, p); err != nil {
		s.discard(p.StorageKey)
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) ([]Photo, error) {
	p, err := s.store(raw)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, p)
	if err != nil {
		s.discard(p.StorageKey)
		return nil, err
	}
	s.discard(oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]Photo, error) {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.discard(key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) ([]Photo, error) {
	seen := map[string]bool{}
	for _, id := range photoIDs {
		if seen[id] {
			return nil, ErrInvalidOrder
		}
		seen[id] = true
	}
	if err := s.repo.Reorder(ctx, userID, photoIDs); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// DeleteStoredFiles removes files for keys whose DB rows are already gone (account deletion).
func (s *Service) DeleteStoredFiles(keys []string) {
	for _, k := range keys {
		s.discard(k)
	}
}
