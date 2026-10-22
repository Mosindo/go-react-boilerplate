package photos

import (
	"context"
	"crypto/md5" // #nosec G501 -- cache-busting label, not a security primitive
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
)

type Service struct {
	repo    Repository
	storage *Storage
}

func NewService(repo Repository, storage *Storage) *Service {
	return &Service{repo: repo, storage: storage}
}

func (s *Service) List(ctx context.Context, userID string) ([]profiles.PhotoRef, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Upload(ctx context.Context, userID string, file io.Reader) (profiles.PhotoRef, error) {
	// Cheap early check; the database constraint is the authority.
	existing, err := s.repo.List(ctx, userID)
	if err != nil {
		return profiles.PhotoRef{}, err
	}
	if len(existing) >= MaxPhotos {
		return profiles.PhotoRef{}, ErrLimitReached
	}

	img, key, err := s.processAndStore(file)
	if err != nil {
		return profiles.PhotoRef{}, err
	}
	p, err := s.repo.Add(ctx, userID, key, img.Width, img.Height, len(img.Full))
	if err != nil {
		s.storage.Remove(key)
		return profiles.PhotoRef{}, err
	}
	return toRef(p), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, file io.Reader) (profiles.PhotoRef, error) {
	img, key, err := s.processAndStore(file)
	if err != nil {
		return profiles.PhotoRef{}, err
	}
	oldKey, p, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Full))
	if err != nil {
		s.storage.Remove(key)
		return profiles.PhotoRef{}, err
	}
	s.storage.Remove(oldKey)
	return toRef(p), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.storage.Remove(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]profiles.PhotoRef, error) {
	for _, id := range ids {
		if !profiles.IsUUID(id) {
			return nil, platformerrors.Invalid("photoIds", "identifiant de photo invalide")
		}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

// SetPrimary moves a photo to the first position.
func (s *Service) SetPrimary(ctx context.Context, userID, photoID string) ([]profiles.PhotoRef, error) {
	current, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := []string{photoID}
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
	return s.Reorder(ctx, userID, ids)
}

// Open returns the image file for viewerID if they are allowed to see it.
func (s *Service) Open(ctx context.Context, viewerID, photoID, variant string) (*os.File, error) {
	p, err := s.repo.GetVisible(ctx, viewerID, photoID)
	if err != nil {
		return nil, err
	}
	f, err := s.storage.Open(p.StorageKey, variant)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *Service) processAndStore(file io.Reader) (ProcessedImage, string, error) {
	img, err := ProcessImage(file)
	if err != nil {
		return ProcessedImage{}, "", err
	}
	key, err := newKey()
	if err != nil {
		return ProcessedImage{}, "", err
	}
	if err := s.storage.Save(key, img.Full, img.Thumb); err != nil {
		s.storage.Remove(key)
		return ProcessedImage{}, "", err
	}
	return img, key, nil
}

func newKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	name := hex.EncodeToString(buf)
	return name[:2] + "/" + name + ".jpg", nil
}

func toRef(p Photo) profiles.PhotoRef {
	sum := md5.Sum([]byte(p.StorageKey))
	return profiles.PhotoRef{ID: p.ID, Position: p.Position, Width: p.Width, Height: p.Height, Version: hex.EncodeToString(sum[:])[:8]}
}
