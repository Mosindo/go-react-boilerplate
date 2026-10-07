package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
)

var (
	ErrPhotoNotFound = platformerrors.NotFound("photo not found")
	ErrPhotoLimit    = platformerrors.Validation("you can upload at most 6 photos")
)

type Service struct {
	repo    *Repository
	storage Storage
	signer  *media.Signer
}

func NewService(repo *Repository, storage Storage, signer *media.Signer) *Service {
	return &Service{repo: repo, storage: storage, signer: signer}
}

func (s *Service) toPhoto(row Row) media.Photo {
	return media.Photo{ID: row.ID, URL: s.signer.URL(row.ID), Position: row.Position}
}

func newKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	name := hex.EncodeToString(buf)
	return "photos/" + name[:2] + "/" + name + ".jpg", nil
}

func (s *Service) List(ctx context.Context, userID string) ([]media.Photo, error) {
	rows, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]media.Photo, 0, len(rows))
	for _, row := range rows {
		out = append(out, s.toPhoto(row))
	}
	return out, nil
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (media.Photo, error) {
	img, err := processImage(raw)
	if err != nil {
		return media.Photo{}, err
	}
	key, err := newKey()
	if err != nil {
		return media.Photo{}, err
	}
	if err := s.storage.Put(key, img.Data); err != nil {
		return media.Photo{}, err
	}
	row, err := s.repo.Add(ctx, userID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.remove(key)
		if errors.Is(err, ErrRepoLimit) {
			return media.Photo{}, ErrPhotoLimit
		}
		return media.Photo{}, err
	}
	return s.toPhoto(row), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (media.Photo, error) {
	img, err := processImage(raw)
	if err != nil {
		return media.Photo{}, err
	}
	key, err := newKey()
	if err != nil {
		return media.Photo{}, err
	}
	if err := s.storage.Put(key, img.Data); err != nil {
		return media.Photo{}, err
	}
	row, oldKey, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.remove(key)
		if errors.Is(err, ErrRepoNotFound) {
			return media.Photo{}, ErrPhotoNotFound
		}
		return media.Photo{}, err
	}
	s.remove(oldKey)
	return s.toPhoto(row), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrPhotoNotFound
		}
		return err
	}
	s.remove(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]media.Photo, error) {
	seen := make(map[string]struct{}, len(ids))
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		if _, dup := seen[id]; dup {
			return nil, platformerrors.Validation("photoIds contains duplicates")
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if len(normalized) == 0 {
		return nil, platformerrors.Validation("photoIds is required")
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return nil, platformerrors.Validation("photoIds must list all of your photos exactly once")
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open returns the image stream for a photo id (the caller has already verified the signed URL).
func (s *Service) Open(ctx context.Context, photoID string) (io.ReadSeekCloser, error) {
	key, err := s.repo.KeyByID(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	f, err := s.storage.Open(key)
	if err != nil {
		return nil, ErrPhotoNotFound
	}
	return f, nil
}

// KeysForUser and Remove implement auth.MediaCleaner for account deletion.
func (s *Service) KeysForUser(ctx context.Context, userID string) ([]string, error) {
	return s.repo.KeysForUser(ctx, userID)
}

func (s *Service) Remove(keys []string) {
	for _, key := range keys {
		s.remove(key)
	}
}

func (s *Service) remove(key string) {
	if err := s.storage.Delete(key); err != nil {
		log.Printf(`{"event":"photo_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}
