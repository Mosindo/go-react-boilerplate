package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"

	"example.com/api/internal/platform/storage"
)

var ErrTooBig = errors.New("photos: file too large")

type Service struct {
	repo  Repository
	store storage.Store
}

func NewService(repo Repository, store storage.Store) *Service {
	return &Service{repo: repo, store: store}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toPhotos(stored), nil
}

func toPhotos(stored []StoredPhoto) []Photo {
	out := make([]Photo, len(stored))
	for i, p := range stored {
		out[i] = toPhoto(p)
	}
	return out
}

// prepare reads at most MaxUploadBytes, sanitises the image and stores the result.
func (s *Service) prepare(ctx context.Context, src io.Reader) (key string, img ProcessedImage, err error) {
	raw, err := io.ReadAll(io.LimitReader(src, MaxUploadBytes+1))
	if err != nil {
		return "", ProcessedImage{}, err
	}
	if len(raw) > MaxUploadBytes {
		return "", ProcessedImage{}, ErrTooBig
	}
	img, err = Process(raw)
	if err != nil {
		return "", ProcessedImage{}, err
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", ProcessedImage{}, err
	}
	key = hex.EncodeToString(buf) + ".jpg"
	if err := s.store.Put(ctx, key, bytesReader(img.Data)); err != nil {
		return "", ProcessedImage{}, err
	}
	return key, img, nil
}

func (s *Service) Upload(ctx context.Context, userID string, src io.Reader) (Photo, error) {
	// Fail fast on the photo limit before spending CPU on image processing.
	existing, err := s.repo.List(ctx, userID)
	if err != nil {
		return Photo{}, err
	}
	if len(existing) >= MaxPhotosPerUser {
		return Photo{}, ErrLimit
	}
	key, img, err := s.prepare(ctx, src)
	if err != nil {
		return Photo{}, err
	}
	stored, err := s.repo.Insert(ctx, userID, key, "image/jpeg", len(img.Data), img.Width, img.Height)
	if err != nil {
		s.discard(key)
		return Photo{}, err
	}
	return toPhoto(stored), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, src io.Reader) (Photo, error) {
	key, img, err := s.prepare(ctx, src)
	if err != nil {
		return Photo{}, err
	}
	oldKey, stored, err := s.repo.Replace(ctx, userID, photoID, key, len(img.Data), img.Width, img.Height)
	if err != nil {
		s.discard(key)
		return Photo{}, err
	}
	s.discard(oldKey)
	return toPhoto(stored), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.discard(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// SetMain moves one photo to position 0, keeping the others in their relative order.
func (s *Service) SetMain(ctx context.Context, userID, photoID string) ([]Photo, error) {
	current, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(current))
	found := false
	for _, p := range current {
		if p.ID == photoID {
			found = true
			continue
		}
		ids = append(ids, p.ID)
	}
	if !found {
		return nil, ErrNotFound
	}
	return s.Reorder(ctx, userID, append([]string{photoID}, ids...))
}

// Open returns the bytes of a photo the viewer is allowed to see.
func (s *Service) Open(ctx context.Context, viewerID, photoID string) (StoredPhoto, io.ReadSeekCloser, error) {
	p, err := s.repo.GetVisible(ctx, viewerID, photoID)
	if err != nil {
		return StoredPhoto{}, nil, err
	}
	f, err := s.store.Open(ctx, p.StorageKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return StoredPhoto{}, nil, ErrNotFound
		}
		return StoredPhoto{}, nil, err
	}
	return p, f, nil
}

// PurgeUser deletes the stored files of a user. Database rows are removed by
// the account deletion cascade. Failures are logged, not fatal: an orphaned file
// is unreachable (no row references it) and must not block account deletion.
func (s *Service) PurgeUser(ctx context.Context, userID string) error {
	keys, err := s.repo.StorageKeys(ctx, userID)
	if err != nil {
		return err
	}
	for _, k := range keys {
		s.discard(k)
	}
	return nil
}

func (s *Service) discard(key string) {
	if key == "" {
		return
	}
	if err := s.store.Delete(context.Background(), key); err != nil {
		log.Printf(`{"event":"photo_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}
