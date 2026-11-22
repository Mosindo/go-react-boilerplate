package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"slices"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/media"
)

type Service struct {
	repo    Repository
	storage media.Storage
	signer  *media.Signer
	now     func() time.Time
}

func NewService(repo Repository, storage media.Storage, signer *media.Signer) *Service {
	return &Service{repo: repo, storage: storage, signer: signer, now: time.Now}
}

func (s *Service) List(ctx context.Context, userID string) ([]profiles.Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]profiles.Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, profiles.Photo{ID: p.ID, URL: s.signer.PhotoURL(p.ID, now), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out, nil
}

func newStorageKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ".jpg", nil
}

// store sanitises and persists an upload, returning its key.
func (s *Service) store(raw []byte) (string, media.ProcessedImage, error) {
	img, err := media.ProcessUpload(raw)
	if err != nil {
		return "", media.ProcessedImage{}, err
	}
	key, err := newStorageKey()
	if err != nil {
		return "", media.ProcessedImage{}, err
	}
	if err := s.storage.Save(key, img.Data); err != nil {
		return "", media.ProcessedImage{}, err
	}
	return key, img, nil
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) ([]profiles.Photo, error) {
	key, img, err := s.store(raw)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Insert(ctx, userID, key, len(img.Data), img.Width, img.Height); err != nil {
		s.removeFile(key)
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) ([]profiles.Photo, error) {
	key, img, err := s.store(raw)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, len(img.Data), img.Width, img.Height)
	if err != nil {
		s.removeFile(key)
		return nil, err
	}
	s.removeFile(oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]profiles.Photo, error) {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.removeFile(key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]profiles.Photo, error) {
	unique := slices.Clone(ids)
	slices.Sort(unique)
	if len(slices.Compact(unique)) != len(ids) {
		return nil, ErrInvalidOrder
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open validates a signed URL and returns the image stream.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (io.ReadSeekCloser, time.Time, error) {
	expires, ok := s.signer.Verify(photoID, exp, sig, s.now())
	if !ok {
		return nil, time.Time{}, ErrPhotoNotFound
	}
	key, err := s.repo.StorageKey(ctx, photoID)
	if err != nil {
		return nil, time.Time{}, err
	}
	f, err := s.storage.Open(key)
	if errors.Is(err, media.ErrNotFound) {
		return nil, time.Time{}, ErrPhotoNotFound
	}
	return f, expires, err
}

// DeleteFiles removes stored blobs (used by account deletion).
func (s *Service) DeleteFiles(keys []string) {
	for _, k := range keys {
		s.removeFile(k)
	}
}

func (s *Service) removeFile(key string) {
	if key == "" {
		return
	}
	if err := s.storage.Delete(key); err != nil {
		log.Printf(`{"event":"photo_file_cleanup_failed","error":%q}`, err.Error())
	}
}
