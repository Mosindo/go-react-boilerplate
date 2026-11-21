package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo     Repository
	store    storage.Storage
	signer   *media.Signer
	profiles *profiles.Service
}

func NewService(repo Repository, store storage.Storage, signer *media.Signer, profilesService *profiles.Service) *Service {
	return &Service{repo: repo, store: store, signer: signer, profiles: profilesService}
}

func randomKey(userID string) (string, string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	base := "photos/" + userID + "/" + hex.EncodeToString(buf)
	return base + ".jpg", base + "_thumb.jpg", nil
}

// store processes and writes both variants, returning the record to persist.
func (s *Service) storeImage(ctx context.Context, userID string, data []byte) (StoredPhoto, error) {
	processed, err := ProcessImage(data)
	if err != nil {
		return StoredPhoto{}, err
	}
	key, thumbKey, err := randomKey(userID)
	if err != nil {
		return StoredPhoto{}, err
	}
	if err := s.store.Put(ctx, key, processed.Full); err != nil {
		return StoredPhoto{}, err
	}
	if err := s.store.Put(ctx, thumbKey, processed.Thumb); err != nil {
		s.removeFiles(ctx, StoredPhoto{StorageKey: key})
		return StoredPhoto{}, err
	}
	return StoredPhoto{
		UserID:          userID,
		StorageKey:      key,
		ThumbStorageKey: thumbKey,
		Width:           processed.Width,
		Height:          processed.Height,
		SizeBytes:       len(processed.Full),
	}, nil
}

func (s *Service) removeFiles(ctx context.Context, p StoredPhoto) {
	for _, key := range []string{p.StorageKey, p.ThumbStorageKey} {
		if key == "" {
			continue
		}
		if err := s.store.Delete(ctx, key); err != nil {
			log.Printf(`{"event":"photo_file_delete_failed","error":%q}`, err.Error())
		}
	}
}

func (s *Service) List(ctx context.Context, userID string) ([]profiles.PhotoResponse, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.responses(stored), nil
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) ([]profiles.PhotoResponse, error) {
	record, err := s.storeImage(ctx, userID, data)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Insert(ctx, record, profiles.MaxPhotos); err != nil {
		s.removeFiles(ctx, record)
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) ([]profiles.PhotoResponse, error) {
	record, err := s.storeImage(ctx, userID, data)
	if err != nil {
		return nil, err
	}
	old, _, err := s.repo.Replace(ctx, userID, photoID, record)
	if err != nil {
		s.removeFiles(ctx, record)
		return nil, err
	}
	s.removeFiles(ctx, old)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]profiles.PhotoResponse, error) {
	deleted, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.removeFiles(ctx, deleted)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) ([]profiles.PhotoResponse, error) {
	seen := map[string]struct{}{}
	for _, id := range photoIDs {
		if _, dup := seen[id]; dup {
			return nil, ErrInvalidOrder
		}
		seen[id] = struct{}{}
	}
	if err := s.repo.Reorder(ctx, userID, photoIDs); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open returns the file for a signed media URL.
func (s *Service) Open(ctx context.Context, photoID string, variant media.Variant, exp, sig string) (io.ReadSeekCloser, error) {
	if !s.signer.Verify(photoID, variant, exp, sig) {
		return nil, ErrInvalidSignature
	}
	photo, err := s.repo.GetByID(ctx, photoID)
	if err != nil {
		return nil, err
	}
	key := photo.StorageKey
	if variant == media.VariantThumb {
		key = photo.ThumbStorageKey
	}
	f, err := s.store.Open(ctx, key)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *Service) responses(stored []StoredPhoto) []profiles.PhotoResponse {
	photos := make([]profiles.Photo, 0, len(stored))
	for _, p := range stored {
		photos = append(photos, profiles.Photo{ID: p.ID, Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return s.profiles.PhotoResponses(photos)
}
