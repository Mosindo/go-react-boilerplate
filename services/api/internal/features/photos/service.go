package photos

import (
	"context"
	"log"
	"strings"

	"example.com/api/internal/platform/apperr"
	"example.com/api/internal/platform/storage"
)

var (
	ErrPhotoNotFound = apperr.NotFound("photo_not_found", "photo not found")
	ErrTooManyPhotos = apperr.Conflict("too_many_photos", "you can have at most 6 photos")
	ErrInvalidOrder  = apperr.BadRequest("invalid_photo_order", "the order must list each of your photos exactly once")
)

type Service struct {
	repo   Repository
	store  storage.Store
	signer *storage.URLSigner
}

func NewService(repo Repository, store storage.Store, signer *storage.URLSigner) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string) ([]PhotoResponse, error) {
	photos, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PhotoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoResponse{ID: p.ID, URL: s.signer.Sign(p.StorageKey), Width: p.Width, Height: p.Height, Position: p.Position})
	}
	return out, nil
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) ([]PhotoResponse, error) {
	stored, err := s.storeImage(ctx, data)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Add(ctx, userID, stored); err != nil {
		s.discard(ctx, stored.StorageKey)
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) ([]PhotoResponse, error) {
	stored, err := s.storeImage(ctx, data)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, stored)
	if err != nil {
		s.discard(ctx, stored.StorageKey)
		return nil, err
	}
	s.discard(ctx, oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]PhotoResponse, error) {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.discard(ctx, key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) ([]PhotoResponse, error) {
	normalized := make([]string, len(photoIDs))
	for i, id := range photoIDs {
		normalized[i] = strings.ToLower(id)
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) storeImage(ctx context.Context, data []byte) (Photo, error) {
	img, err := ProcessImage(data)
	if err != nil {
		return Photo{}, err
	}
	key, err := storage.NewKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.store.Put(ctx, key, img.Data); err != nil {
		return Photo{}, err
	}
	return Photo{StorageKey: key, Width: img.Width, Height: img.Height, SizeBytes: len(img.Data)}, nil
}

func (s *Service) discard(ctx context.Context, key string) {
	if key == "" {
		return
	}
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"photo_file_delete_error","key":%q,"error":%q}`, key, err.Error())
	}
}
