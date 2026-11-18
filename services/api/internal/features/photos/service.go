package photos

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"strings"

	"example.com/api/internal/platform/media"
)

var (
	ErrPhotoNotFound = errors.New("photo not found")
	ErrTooManyPhotos = errors.New("you can upload up to 6 photos")
	ErrInvalidOrder  = errors.New("order must list every photo exactly once")
)

type Signer interface {
	PhotoURL(photoID string) string
	Verify(photoID, exp, sig string) bool
}

type Service struct {
	repo    Repository
	storage media.Storage
	signer  Signer
}

func NewService(repo Repository, storage media.Storage, signer Signer) *Service {
	return &Service{repo: repo, storage: storage, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, Photo{ID: p.ID, Position: p.Position, URL: s.signer.PhotoURL(p.ID), Width: p.Width, Height: p.Height})
	}
	return out, nil
}

// store sanitises the upload and writes it to storage, returning its key.
func (s *Service) store(r io.Reader) (string, media.ProcessedImage, error) {
	img, err := media.ProcessUpload(r)
	if err != nil {
		return "", media.ProcessedImage{}, err
	}
	key, err := media.NewKey()
	if err != nil {
		return "", media.ProcessedImage{}, err
	}
	if err := s.storage.Save(key, bytes.NewReader(img.Data)); err != nil {
		return "", media.ProcessedImage{}, err
	}
	return key, img, nil
}

func (s *Service) Upload(ctx context.Context, userID string, r io.Reader) ([]Photo, error) {
	existing, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= MaxPhotos {
		return nil, ErrTooManyPhotos
	}
	key, img, err := s.store(r)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Insert(ctx, userID, key, img.Width, img.Height, len(img.Data)); err != nil {
		s.removeBlob(key)
		if errors.Is(err, ErrRepositoryFull) {
			return nil, ErrTooManyPhotos
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, r io.Reader) ([]Photo, error) {
	key, img, err := s.store(r)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.removeBlob(key)
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	s.removeBlob(oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]Photo, error) {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	s.removeBlob(key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) ([]Photo, error) {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(photoIDs))
	for _, id := range photoIDs {
		id = strings.ToLower(strings.TrimSpace(id))
		if _, dup := seen[id]; dup {
			return nil, ErrInvalidOrder
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		if errors.Is(err, ErrRepositoryOrder) {
			return nil, ErrInvalidOrder
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open returns the stored file behind a valid signed URL.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (string, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return "", ErrPhotoNotFound
	}
	key, err := s.repo.GetStorageKey(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return "", ErrPhotoNotFound
		}
		return "", err
	}
	return s.storage.Path(key)
}

func (s *Service) removeBlob(key string) {
	if err := s.storage.Delete(key); err != nil {
		log.Printf(`{"event":"photo_blob_delete_error","error":%q}`, err.Error())
	}
}
