package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"

	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo  Repository
	store storage.Storage
}

func NewService(repo Repository, store storage.Storage) *Service {
	return &Service{repo: repo, store: store}
}

func newKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + storedExtension, nil
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, p.Photo)
	}
	return out, nil
}

// Upload validates, re-encodes and stores a new photo at the next position.
func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (Photo, error) {
	processed, err := Process(raw)
	if err != nil {
		return Photo{}, err
	}
	key, err := newKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.store.Put(ctx, key, processed.Data); err != nil {
		return Photo{}, err
	}
	stored, err := s.repo.Add(ctx, userID, key, len(processed.Data), processed.Width, processed.Height)
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return Photo{}, err
	}
	return stored.Photo, nil
}

// Replace swaps the content of an existing photo, keeping its position.
func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (Photo, error) {
	processed, err := Process(raw)
	if err != nil {
		return Photo{}, err
	}
	key, err := newKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.store.Put(ctx, key, processed.Data); err != nil {
		return Photo{}, err
	}
	stored, old, err := s.repo.Replace(ctx, userID, photoID, key, len(processed.Data), processed.Width, processed.Height)
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return Photo{}, err
	}
	_ = s.store.Delete(ctx, old)
	return stored.Photo, nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	_ = s.store.Delete(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}

// Open returns the photo bytes if viewerID is allowed to see them.
func (s *Service) Open(ctx context.Context, viewerID, photoID string) (io.ReadSeekCloser, string, error) {
	key, err := s.repo.Viewable(ctx, viewerID, photoID)
	if err != nil {
		return nil, "", err
	}
	f, err := s.store.Open(ctx, key)
	if err != nil {
		return nil, "", ErrNotFound
	}
	return f, key, nil
}
