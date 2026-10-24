package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"strings"

	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo  Repository
	store storage.Store
	// sem bounds concurrent image decoding, which is the memory-heavy part of an upload.
	sem chan struct{}
}

func NewService(repo Repository, store storage.Store) *Service {
	return &Service{repo: repo, store: store, sem: make(chan struct{}, 4)}
}

func newKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}

func thumbKey(key string) string { return strings.TrimSuffix(key, ".jpg") + "_t.jpg" }

func (s *Service) prepare(ctx context.Context, data []byte) (processed, string, error) {
	if len(data) == 0 {
		return processed{}, "", ErrEmptyUpload
	}
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return processed{}, "", ctx.Err()
	}
	img, err := processImage(data)
	if err != nil {
		return processed{}, "", err
	}
	key, err := newKey()
	if err != nil {
		return processed{}, "", err
	}
	if err := s.store.Put(key, img.Full); err != nil {
		return processed{}, "", err
	}
	if err := s.store.Put(thumbKey(key), img.Thumb); err != nil {
		_ = s.store.Delete(key)
		return processed{}, "", err
	}
	return img, key, nil
}

func (s *Service) discard(key string) {
	for _, k := range []string{key, thumbKey(key)} {
		if err := s.store.Delete(k); err != nil {
			log.Printf(`{"event":"photo_cleanup_failed","key":%q,"error":%q}`, k, err.Error())
		}
	}
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) (Photo, error) {
	img, key, err := s.prepare(ctx, data)
	if err != nil {
		return Photo{}, err
	}
	p, err := s.repo.Add(ctx, userID, key, len(img.Full), img.Width, img.Height)
	if err != nil {
		s.discard(key)
		return Photo{}, err
	}
	return Photo{ID: p.ID, Position: p.Position, Width: p.Width, Height: p.Height}, nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) (Photo, error) {
	img, key, err := s.prepare(ctx, data)
	if err != nil {
		return Photo{}, err
	}
	p, oldKey, err := s.repo.Replace(ctx, userID, photoID, key, len(img.Full), img.Width, img.Height)
	if err != nil {
		s.discard(key)
		return Photo{}, err
	}
	s.discard(oldKey)
	return Photo{ID: p.ID, Position: p.Position, Width: p.Width, Height: p.Height}, nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.discard(key)
	return nil
}

// Open returns the image stream when the viewer is allowed to see the photo.
func (s *Service) Open(ctx context.Context, viewerID, photoID string, thumb bool) (io.ReadSeekCloser, error) {
	key, err := s.repo.VisibleKey(ctx, viewerID, photoID)
	if err != nil {
		return nil, err
	}
	if thumb {
		key = thumbKey(key)
	}
	f, err := s.store.Open(key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return f, err
}
