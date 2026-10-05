package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"

	"example.com/api/internal/platform/signing"
)

type Service struct {
	repo    Repository
	storage Storage
	signer  *signing.Signer
	// cpu bounds concurrent image processing so uploads cannot exhaust memory or CPU.
	cpu chan struct{}
}

func NewService(repo Repository, storage Storage, signer *signing.Signer) *Service {
	return &Service{repo: repo, storage: storage, signer: signer, cpu: make(chan struct{}, 3)}
}

func newKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}

func (s *Service) toPhoto(p storedPhoto) Photo {
	return Photo{ID: p.ID, URL: s.signer.PhotoPath(p.ID), Position: p.Position, Width: p.Width, Height: p.Height, CreatedAt: p.CreatedAt}
}

func (s *Service) process(ctx context.Context, raw []byte) (processed, string, error) {
	select {
	case s.cpu <- struct{}{}:
		defer func() { <-s.cpu }()
	case <-ctx.Done():
		return processed{}, "", ctx.Err()
	}
	img, err := processImage(raw)
	if err != nil {
		return processed{}, "", err
	}
	key, err := newKey()
	if err != nil {
		return processed{}, "", err
	}
	if err := s.storage.Put(ctx, key, img.Data); err != nil {
		return processed{}, "", err
	}
	return img, key, nil
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

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (Photo, error) {
	img, key, err := s.process(ctx, raw)
	if err != nil {
		return Photo{}, err
	}
	stored, err := s.repo.Insert(ctx, userID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.deleteQuietly(key)
		return Photo{}, err
	}
	return s.toPhoto(stored), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (Photo, error) {
	img, key, err := s.process(ctx, raw)
	if err != nil {
		return Photo{}, err
	}
	oldKey, stored, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.deleteQuietly(key)
		return Photo{}, err
	}
	s.deleteQuietly(oldKey)
	return s.toPhoto(stored), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.deleteQuietly(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}

// Open returns the bytes of a photo if the signed URL parameters are valid.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (io.ReadSeekCloser, error) {
	if !s.signer.VerifyPhoto(photoID, exp, sig) {
		return nil, ErrNotFound
	}
	stored, err := s.repo.Get(ctx, photoID)
	if err != nil {
		return nil, err
	}
	f, err := s.storage.Open(ctx, stored.StorageKey)
	if errors.Is(err, ErrStorageNotFound) {
		return nil, ErrNotFound
	}
	return f, err
}

// Prepare implements auth.AccountEraser: it remembers the user's files before the rows are cascaded away
// and returns the function that deletes them from storage once the account is gone.
func (s *Service) Prepare(ctx context.Context, userID string) (func(), error) {
	keys, err := s.repo.Keys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() {
		for _, k := range keys {
			s.deleteQuietly(k)
		}
	}, nil
}

func (s *Service) deleteQuietly(key string) {
	if key == "" {
		return
	}
	if err := s.storage.Delete(context.Background(), key); err != nil {
		log.Printf("photos: delete %s: %v", key, err)
	}
}
