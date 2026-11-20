package photos

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"slices"

	"example.com/api/internal/platform/mediaurl"
	"example.com/api/internal/platform/storage"
)

var (
	ErrPhotoNotFound    = errors.New("photo not found")
	ErrPhotoLimit       = errors.New("you can upload up to 6 photos")
	ErrLastPhoto        = errors.New("keep at least one photo; replace it instead")
	ErrInvalidOrder     = errors.New("the new order must list every photo exactly once")
	ErrInvalidSignature = errors.New("invalid or expired media link")
)

type Service struct {
	repo   Repository
	store  storage.Store
	signer *mediaurl.Signer
}

func NewService(repo Repository, store storage.Store, signer *mediaurl.Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

func (s *Service) toPhoto(p StoredPhoto) Photo {
	return Photo{ID: p.ID, UserID: p.UserID, Position: p.Position, URL: s.signer.PhotoURL(p.ID), Width: p.Width, Height: p.Height, CreatedAt: p.CreatedAt}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, s.toPhoto(p))
	}
	return out, nil
}

func (s *Service) storeProcessed(ctx context.Context, raw []byte) (NewPhoto, error) {
	processed, err := processImage(raw)
	if err != nil {
		return NewPhoto{}, err
	}
	key, err := storage.NewKey("jpg")
	if err != nil {
		return NewPhoto{}, err
	}
	if err := s.store.Put(ctx, key, bytes.NewReader(processed.data)); err != nil {
		return NewPhoto{}, err
	}
	return NewPhoto{StorageKey: key, Width: processed.width, Height: processed.height, SizeBytes: len(processed.data)}, nil
}

func (s *Service) discard(ctx context.Context, key string) {
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"photo_cleanup_failed","error":%q}`, err.Error())
	}
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (Photo, error) {
	current, err := s.repo.List(ctx, userID)
	if err != nil {
		return Photo{}, err
	}
	if len(current) >= MaxPhotosPerUser {
		return Photo{}, ErrPhotoLimit
	}
	photo, err := s.storeProcessed(ctx, raw)
	if err != nil {
		return Photo{}, err
	}
	stored, err := s.repo.Insert(ctx, userID, photo, MaxPhotosPerUser)
	if err != nil {
		s.discard(ctx, photo.StorageKey)
		if errors.Is(err, ErrRepositoryLimit) {
			return Photo{}, ErrPhotoLimit
		}
		return Photo{}, err
	}
	return s.toPhoto(stored), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (Photo, error) {
	photo, err := s.storeProcessed(ctx, raw)
	if err != nil {
		return Photo{}, err
	}
	stored, oldKey, err := s.repo.Replace(ctx, userID, photoID, photo)
	if err != nil {
		s.discard(ctx, photo.StorageKey)
		if errors.Is(err, ErrRepositoryNotFound) {
			return Photo{}, ErrPhotoNotFound
		}
		return Photo{}, err
	}
	s.discard(ctx, oldKey)
	return s.toPhoto(stored), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	switch {
	case errors.Is(err, ErrRepositoryNotFound):
		return ErrPhotoNotFound
	case errors.Is(err, ErrRepositoryLast):
		return ErrLastPhoto
	case err != nil:
		return err
	}
	s.discard(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, orderedIDs []string) ([]Photo, error) {
	unique := slices.Compact(slices.Sorted(slices.Values(orderedIDs)))
	if len(unique) != len(orderedIDs) {
		return nil, ErrInvalidOrder
	}
	if err := s.repo.Reorder(ctx, userID, orderedIDs); err != nil {
		if errors.Is(err, ErrRepositoryMismatch) {
			return nil, ErrInvalidOrder
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// OpenSigned returns the image bytes for a valid signed URL.
func (s *Service) OpenSigned(ctx context.Context, photoID, expires, signature string) (io.ReadCloser, error) {
	if !s.signer.Verify(photoID, expires, signature) {
		return nil, ErrInvalidSignature
	}
	key, err := s.repo.StorageKey(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	rc, err := s.store.Open(ctx, key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrPhotoNotFound
	}
	return rc, err
}

// StorageKeysForUser and DeleteStoredObjects implement auth.AccountPurger.
func (s *Service) StorageKeysForUser(ctx context.Context, userID string) ([]string, error) {
	return s.repo.StorageKeysForUser(ctx, userID)
}

func (s *Service) DeleteStoredObjects(ctx context.Context, keys []string) {
	for _, key := range keys {
		s.discard(ctx, key)
	}
}
