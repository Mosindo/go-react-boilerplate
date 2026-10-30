package photos

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"log"

	"example.com/api/internal/platform/imaging"
	"example.com/api/internal/platform/storage"
)

type URLSigner interface {
	PhotoPath(photoID string) string
	Verify(id, expRaw, sig string) bool
}

type Service struct {
	repo   Repository
	store  storage.Store
	signer URLSigner
}

func NewService(repo Repository, store storage.Store, signer URLSigner) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

func (s *Service) toPhotos(stored []storedPhoto) []Photo {
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, Photo{ID: p.ID, URL: s.signer.PhotoPath(p.ID), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.toPhotos(stored), nil
}

// Upload validates and normalizes the image, stores it, then records it.
func (s *Service) Upload(ctx context.Context, userID string, r io.Reader) ([]Photo, error) {
	res, key, err := s.prepare(r)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Add(ctx, userID, key, res.Width, res.Height, len(res.Data)); err != nil {
		s.discard(key)
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, r io.Reader) ([]Photo, error) {
	res, key, err := s.prepare(r)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, res.Width, res.Height, len(res.Data))
	if err != nil {
		s.discard(key)
		return nil, err
	}
	s.discard(oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]Photo, error) {
	key, err := s.repo.Remove(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.discard(key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open returns the image bytes for a signed URL.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (io.ReadSeekCloser, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return nil, ErrNotFound
	}
	key, err := s.repo.KeyByID(ctx, photoID)
	if err != nil {
		return nil, err
	}
	f, err := s.store.Open(key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return f, err
}

// StorageKeys lists the files owned by a user, for cleanup after account deletion.
func (s *Service) StorageKeys(ctx context.Context, userID string) ([]string, error) {
	return s.repo.KeysByUser(ctx, userID)
}

func (s *Service) RemoveFiles(keys []string) {
	for _, key := range keys {
		s.discard(key)
	}
}

func (s *Service) prepare(r io.Reader) (imaging.Result, string, error) {
	res, err := imaging.Process(r)
	if err != nil {
		return imaging.Result{}, "", err
	}
	random := make([]byte, 17)
	if _, err := rand.Read(random); err != nil {
		return imaging.Result{}, "", err
	}
	key := storage.NewKey(random)
	if err := s.store.Put(key, res.Data); err != nil {
		return imaging.Result{}, "", err
	}
	return res, key, nil
}

func (s *Service) discard(key string) {
	if err := s.store.Delete(key); err != nil {
		log.Printf(`{"event":"photo_cleanup_failed","error":%q}`, err.Error())
	}
}
