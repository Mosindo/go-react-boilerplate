package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/urlsign"
)

var (
	ErrPhotoNotFound    = apperr.New(http.StatusNotFound, "photo_not_found", "photo not found")
	ErrPhotoLimit       = apperr.New(http.StatusConflict, "photo_limit", "you can add up to 6 photos")
	ErrUnsupportedImage = apperr.Validation("unsupported_image", "only JPEG and PNG images are accepted")
	ErrImageTooSmall    = apperr.Validation("image_too_small", "image must be at least 200x200 pixels")
	ErrImageTooLarge    = apperr.Validation("image_too_large", "image is too large")
	ErrInvalidOrder     = apperr.Validation("invalid_order", "the order must list each of your photos exactly once")
)

// MaxUploadBytes bounds the raw upload size (clients compress beforehand).
const MaxUploadBytes = 10 << 20

type Service struct {
	repo    Repository
	storage storage.Storage
	signer  *urlsign.Signer
}

func NewService(repo Repository, s storage.Storage, signer *urlsign.Signer) *Service {
	return &Service{repo: repo, storage: s, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.toPhotos(stored), nil
}

// ListForUsers returns the photos of many users keyed by user id.
func (s *Service) ListForUsers(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	stored, err := s.repo.ListByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]Photo, len(stored))
	for userID, list := range stored {
		out[userID] = s.toPhotos(list)
	}
	return out, nil
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) (Photo, error) {
	key, processed, w, h, err := s.store(ctx, data)
	if err != nil {
		return Photo{}, err
	}
	p, err := s.repo.Insert(ctx, userID, key, w, h, len(processed))
	if err != nil {
		s.discard(ctx, key)
		if errors.Is(err, errRepoLimit) {
			return Photo{}, ErrPhotoLimit
		}
		return Photo{}, err
	}
	return s.toPhoto(p), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) (Photo, error) {
	key, processed, w, h, err := s.store(ctx, data)
	if err != nil {
		return Photo{}, err
	}
	oldKey, p, err := s.repo.Replace(ctx, userID, photoID, key, w, h, len(processed))
	if err != nil {
		s.discard(ctx, key)
		if errors.Is(err, errRepoNotFound) {
			return Photo{}, ErrPhotoNotFound
		}
		return Photo{}, err
	}
	s.discard(ctx, oldKey)
	return s.toPhoto(p), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if errors.Is(err, errRepoNotFound) {
		return ErrPhotoNotFound
	}
	if err != nil {
		return err
	}
	s.discard(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	seen := make(map[string]struct{}, len(ids))
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		if _, dup := seen[id]; dup {
			return nil, ErrInvalidOrder
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		if errors.Is(err, errRepoMismatch) {
			return nil, ErrInvalidOrder
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// FilePath verifies a signed photo URL and returns the file to serve.
func (s *Service) FilePath(ctx context.Context, photoID, exp, sig string) (string, error) {
	key, err := s.repo.StorageKey(ctx, photoID)
	if errors.Is(err, errRepoNotFound) {
		return "", ErrPhotoNotFound
	}
	if err != nil {
		return "", err
	}
	// The signature covers the stored file version: replacing or deleting a
	// photo invalidates every URL previously handed out.
	if !s.signer.Verify(signedResource(photoID, key), exp, sig) {
		return "", ErrPhotoNotFound
	}
	return s.storage.Path(key)
}

func signedResource(photoID, storageKey string) string {
	return photoID + ":" + storageKey
}

func (s *Service) store(ctx context.Context, data []byte) (string, []byte, int, int, error) {
	processed, w, h, err := processImage(data)
	switch {
	case errors.Is(err, errUnsupportedImage):
		return "", nil, 0, 0, ErrUnsupportedImage
	case errors.Is(err, errImageTooSmall):
		return "", nil, 0, 0, ErrImageTooSmall
	case errors.Is(err, errImageTooLarge):
		return "", nil, 0, 0, ErrImageTooLarge
	case err != nil:
		return "", nil, 0, 0, err
	}
	key, err := newStorageKey()
	if err != nil {
		return "", nil, 0, 0, err
	}
	if err := s.storage.Save(ctx, key, processed); err != nil {
		return "", nil, 0, 0, err
	}
	return key, processed, w, h, nil
}

func (s *Service) discard(ctx context.Context, key string) {
	if key == "" {
		return
	}
	if err := s.storage.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"photo_file_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}

func (s *Service) toPhotos(list []storedPhoto) []Photo {
	out := make([]Photo, 0, len(list))
	for _, p := range list {
		out = append(out, s.toPhoto(p))
	}
	return out
}

func (s *Service) toPhoto(p storedPhoto) Photo {
	return Photo{
		ID:       p.ID,
		Position: p.Position,
		Width:    p.Width,
		Height:   p.Height,
		URL:      "/photos/" + p.ID + "?" + s.signer.Query(signedResource(p.ID, p.StorageKey)),
	}
}

func newStorageKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}
