package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"strings"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo   Repository
	store  storage.Store
	signer *media.Signer
}

func NewService(repo Repository, store storage.Store, signer *media.Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string) ([]profiles.Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, profiles.Photo{ID: p.ID, URL: s.signer.PhotoURL(p.ID), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out, nil
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) ([]profiles.Photo, error) {
	img, err := ProcessImage(data)
	if err != nil {
		return nil, err
	}
	key, err := newStorageKey()
	if err != nil {
		return nil, err
	}
	if err := s.store.Put(ctx, key, img.Data); err != nil {
		return nil, err
	}
	if _, err := s.repo.Insert(ctx, userID, key, img.Width, img.Height, len(img.Data)); err != nil {
		s.cleanup(ctx, key)
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) ([]profiles.Photo, error) {
	img, err := ProcessImage(data)
	if err != nil {
		return nil, err
	}
	key, err := newStorageKey()
	if err != nil {
		return nil, err
	}
	if err := s.store.Put(ctx, key, img.Data); err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.cleanup(ctx, key)
		return nil, err
	}
	s.cleanup(ctx, oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]profiles.Photo, error) {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.cleanup(ctx, key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) ([]profiles.Photo, error) {
	normalized := make([]string, 0, len(photoIDs))
	for _, id := range photoIDs {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(id)))
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open returns the file of a photo after the signed URL has been verified.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (io.ReadSeekCloser, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return nil, ErrPhotoNotFound
	}
	key, err := s.repo.GetStorageKey(ctx, photoID)
	if err != nil {
		return nil, err
	}
	f, err := s.store.Open(ctx, key)
	if err != nil {
		return nil, ErrPhotoNotFound
	}
	return f, nil
}

func (s *Service) cleanup(ctx context.Context, key string) {
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"photo_cleanup_failed","storage_key":%q}`, key)
	}
}

func newStorageKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "photos/" + hex.EncodeToString(b) + ".jpg", nil
}
