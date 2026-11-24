package photos

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
)

var (
	ErrPhotoNotFound   = apperr.NotFound("photo_not_found", "photo not found")
	ErrPhotoLimit      = apperr.Conflict("photo_limit_reached", "you can upload up to 6 photos")
	ErrReorderMismatch = apperr.BadRequest("invalid_photo_order", "the order must list each of your photos exactly once")
	ErrInvalidSigned   = apperr.Forbidden("invalid_media_signature", "this link is invalid or expired")
)

type Service struct {
	repo   Repository
	store  storage.Store
	signer *media.Signer
}

func NewService(repo Repository, store storage.Store, signer *media.Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

// ToResponse attaches a signed URL to a photo.
func (s *Service) ToResponse(p Photo) PhotoResponse {
	return PhotoResponse{ID: p.ID, URL: s.signer.PhotoURL(p.ID), Position: p.Position, Width: p.Width, Height: p.Height}
}

func (s *Service) toResponses(list []Photo) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(list))
	for _, p := range list {
		out = append(out, s.ToResponse(p))
	}
	return out
}

func (s *Service) ListMine(ctx context.Context, userID string) ([]PhotoResponse, error) {
	list, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.toResponses(list), nil
}

// ListForUsers returns signed photos grouped by user, in a single query.
func (s *Service) ListForUsers(ctx context.Context, userIDs []string) (map[string][]PhotoResponse, error) {
	list, err := s.repo.ListForUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	grouped := make(map[string][]PhotoResponse, len(userIDs))
	for _, p := range list {
		grouped[p.UserID] = append(grouped[p.UserID], s.ToResponse(p))
	}
	return grouped, nil
}

func (s *Service) storeProcessed(ctx context.Context, userID string, data []byte) (string, ProcessedImage, error) {
	processed, err := ProcessImage(data)
	if err != nil {
		return "", ProcessedImage{}, err
	}
	key, err := storage.NewKey("photos/"+userID, ".jpg")
	if err != nil {
		return "", ProcessedImage{}, err
	}
	if err := s.store.Put(ctx, key, processed.Data); err != nil {
		return "", ProcessedImage{}, err
	}
	return key, processed, nil
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) (PhotoResponse, error) {
	key, processed, err := s.storeProcessed(ctx, userID, data)
	if err != nil {
		return PhotoResponse{}, err
	}
	photo, err := s.repo.Insert(ctx, userID, key, processed.Width, processed.Height, len(processed.Data))
	if err != nil {
		s.deleteObject(ctx, key)
		if errors.Is(err, ErrRepositoryFull) {
			return PhotoResponse{}, ErrPhotoLimit
		}
		return PhotoResponse{}, err
	}
	return s.ToResponse(photo), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) (PhotoResponse, error) {
	key, processed, err := s.storeProcessed(ctx, userID, data)
	if err != nil {
		return PhotoResponse{}, err
	}
	oldKey, photo, err := s.repo.ReplaceFile(ctx, userID, photoID, key, processed.Width, processed.Height, len(processed.Data))
	if err != nil {
		s.deleteObject(ctx, key)
		if errors.Is(err, ErrRepositoryNotFound) {
			return PhotoResponse{}, ErrPhotoNotFound
		}
		return PhotoResponse{}, err
	}
	s.deleteObject(ctx, oldKey)
	return s.ToResponse(photo), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrPhotoNotFound
		}
		return err
	}
	s.deleteObject(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, orderedIDs []string) ([]PhotoResponse, error) {
	seen := make(map[string]struct{}, len(orderedIDs))
	normalized := make([]string, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		id = strings.ToLower(strings.TrimSpace(id))
		if _, dup := seen[id]; dup {
			return nil, ErrReorderMismatch
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		if errors.Is(err, ErrRepositoryMismatch) {
			return nil, ErrReorderMismatch
		}
		return nil, err
	}
	return s.ListMine(ctx, userID)
}

// OpenSigned returns the file behind a signed URL.
func (s *Service) OpenSigned(ctx context.Context, photoID, expires, signature string) (io.ReadSeekCloser, Photo, error) {
	if !s.signer.Verify(photoID, expires, signature) {
		return nil, Photo{}, ErrInvalidSigned
	}
	photo, err := s.repo.Get(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, Photo{}, ErrPhotoNotFound
		}
		return nil, Photo{}, err
	}
	file, err := s.store.Open(ctx, photo.StorageKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, Photo{}, ErrPhotoNotFound
		}
		return nil, Photo{}, err
	}
	return file, photo, nil
}

// StorageKeysForUser and PurgeObjects implement auth.AccountEraser.
func (s *Service) StorageKeysForUser(ctx context.Context, userID string) ([]string, error) {
	return s.repo.StorageKeysForUser(ctx, userID)
}

func (s *Service) PurgeObjects(ctx context.Context, keys []string) {
	for _, key := range keys {
		s.deleteObject(ctx, key)
	}
}

func (s *Service) deleteObject(ctx context.Context, key string) {
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"storage_delete_failed","error":%q}`, err.Error())
	}
}
