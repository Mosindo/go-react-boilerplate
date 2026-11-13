package photos

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"

	"example.com/api/internal/platform/storage"
)

var ErrInvalidSignature = errors.New("invalid or expired media signature")

// Signer issues and verifies short-lived photo URLs.
type Signer interface {
	PhotoURL(photoID string) string
	Verify(photoID, exp, sig string) bool
}

type Service struct {
	repo   Repository
	store  storage.Store
	signer Signer
}

func NewService(repo Repository, store storage.Store, signer Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

func (s *Service) toPhoto(p storedPhoto) Photo {
	return Photo{
		ID:        p.ID,
		URL:       s.signer.PhotoURL(p.ID),
		Position:  p.Position,
		Width:     p.Width,
		Height:    p.Height,
		CreatedAt: p.CreatedAt,
	}
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

func newStorageKey(userID string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// Random, unguessable object names scoped under the owner's prefix so
	// account deletion can remove everything at once.
	return "photos/" + userID + "/" + hex.EncodeToString(buf) + ".jpg", nil
}

func (s *Service) storeProcessed(ctx context.Context, userID string, r io.Reader) (string, ProcessedImage, error) {
	img, err := ProcessUpload(r)
	if err != nil {
		return "", ProcessedImage{}, err
	}
	key, err := newStorageKey(userID)
	if err != nil {
		return "", ProcessedImage{}, err
	}
	if err := s.store.Put(ctx, key, bytes.NewReader(img.Data)); err != nil {
		return "", ProcessedImage{}, err
	}
	return key, img, nil
}

func (s *Service) Upload(ctx context.Context, userID string, r io.Reader) (Photo, error) {
	existing, err := s.repo.List(ctx, userID)
	if err != nil {
		return Photo{}, err
	}
	if len(existing) >= MaxPhotosPerUser {
		return Photo{}, ErrTooManyPhotos
	}

	key, img, err := s.storeProcessed(ctx, userID, r)
	if err != nil {
		return Photo{}, err
	}
	stored, err := s.repo.Insert(ctx, userID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.deleteObject(ctx, key)
		return Photo{}, err
	}
	return s.toPhoto(stored), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, r io.Reader) (Photo, error) {
	key, img, err := s.storeProcessed(ctx, userID, r)
	if err != nil {
		return Photo{}, err
	}
	oldKey, stored, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.deleteObject(ctx, key)
		return Photo{}, err
	}
	s.deleteObject(ctx, oldKey)
	return s.toPhoto(stored), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.deleteObject(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) ([]Photo, error) {
	if err := s.repo.Reorder(ctx, userID, photoIDs); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open verifies the URL signature before streaming the stored JPEG.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (io.ReadCloser, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return nil, ErrInvalidSignature
	}
	key, err := s.repo.StorageKey(ctx, photoID)
	if err != nil {
		return nil, err
	}
	rc, err := s.store.Open(ctx, key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	return rc, nil
}

// DeleteUserMedia implements auth.AccountCleaner.
func (s *Service) DeleteUserMedia(ctx context.Context, userID string) error {
	return s.store.DeletePrefix(ctx, "photos/"+userID)
}

func (s *Service) deleteObject(ctx context.Context, key string) {
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"storage_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}
