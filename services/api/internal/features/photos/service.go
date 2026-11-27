package photos

import (
	"context"
	"errors"
	"io"
	"log"
	"time"

	"example.com/api/internal/platform/storage"
)

// Signer issues and verifies media URLs.
type Signer interface {
	PhotoURL(photoID string) string
	Verify(id, exp, sig string) bool
}

type Service struct {
	repo   Repository
	store  storage.Store
	signer Signer
}

func NewService(repo Repository, store storage.Store, signer Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string) ([]PhotoResponse, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PhotoResponse, 0, len(list))
	for _, p := range list {
		out = append(out, PhotoResponse{ID: p.ID, URL: s.signer.PhotoURL(p.ID), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out, nil
}

func (s *Service) storeProcessed(ctx context.Context, userID string, raw []byte) (string, processedImage, error) {
	img, err := processImage(raw)
	if err != nil {
		return "", processedImage{}, err
	}
	key, err := storage.NewPhotoKey(userID)
	if err != nil {
		return "", processedImage{}, err
	}
	if err := s.store.Put(ctx, key, img.Data); err != nil {
		return "", processedImage{}, err
	}
	return key, img, nil
}

// Upload validates, normalises and stores a new photo at the end of the list.
func (s *Service) Upload(ctx context.Context, userID string, raw []byte) ([]PhotoResponse, error) {
	key, img, err := s.storeProcessed(ctx, userID, raw)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Insert(ctx, Photo{UserID: userID, StorageKey: key, Width: img.Width, Height: img.Height, SizeBytes: len(img.Data)}); err != nil {
		s.removeObject(ctx, key)
		return nil, err
	}
	return s.List(ctx, userID)
}

// Replace swaps the image of an existing photo, keeping its position.
func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) ([]PhotoResponse, error) {
	key, img, err := s.storeProcessed(ctx, userID, raw)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.removeObject(ctx, key)
		return nil, err
	}
	s.removeObject(ctx, oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]PhotoResponse, error) {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.removeObject(ctx, key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) ([]PhotoResponse, error) {
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

// OpenSigned resolves a signed media request.
func (s *Service) OpenSigned(ctx context.Context, photoID, exp, sig string) (io.ReadSeekCloser, time.Time, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return nil, time.Time{}, ErrPhotoNotFound
	}
	key, err := s.repo.StorageKeyByID(ctx, photoID)
	if err != nil {
		return nil, time.Time{}, err
	}
	f, mod, err := s.store.Open(ctx, key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, time.Time{}, ErrPhotoNotFound
		}
		return nil, time.Time{}, err
	}
	return f, mod, nil
}

// CollectUserObjects implements auth.AccountDeletionHook.
func (s *Service) CollectUserObjects(ctx context.Context, userID string) (func(context.Context), error) {
	keys, err := s.repo.StorageKeysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) {
		for _, k := range keys {
			s.removeObject(ctx, k)
		}
	}, nil
}

func (s *Service) removeObject(ctx context.Context, key string) {
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"storage_delete_failed","error":%q}`, err.Error())
	}
}
