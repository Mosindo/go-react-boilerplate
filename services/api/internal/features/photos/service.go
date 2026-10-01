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

type Service struct {
	repo  *Repository
	store storage.Store
}

func NewService(repo *Repository, store storage.Store) *Service {
	return &Service{repo: repo, store: store}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

func newKeys(userID string) (full, thumb string, err error) {
	buf := make([]byte, 16)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	base := "photos/" + userID + "/" + hex.EncodeToString(buf)
	return base + ".jpg", base + "_t.jpg", nil
}

// store processes and persists both variants, returning their keys.
func (s *Service) process(ctx context.Context, userID string, r io.Reader) (full, thumb string, p Processed, err error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return "", "", Processed{}, err
	}
	if len(raw) == 0 {
		return "", "", Processed{}, ErrFileRequired
	}
	if len(raw) > MaxUploadBytes {
		return "", "", Processed{}, ErrFileTooLarge
	}
	p, err = Process(raw)
	if err != nil {
		return "", "", Processed{}, err
	}
	full, thumb, err = newKeys(userID)
	if err != nil {
		return "", "", Processed{}, err
	}
	if err := s.store.Put(ctx, full, p.Full); err != nil {
		return "", "", Processed{}, errors.Join(ErrStorageFailed, err)
	}
	if err := s.store.Put(ctx, thumb, p.Thumb); err != nil {
		_ = s.store.Delete(ctx, full)
		return "", "", Processed{}, errors.Join(ErrStorageFailed, err)
	}
	return full, thumb, p, nil
}

func (s *Service) removeFiles(ctx context.Context, keys ...string) {
	for _, k := range keys {
		if err := s.store.Delete(ctx, k); err != nil {
			log.Printf("photos: could not delete %s: %v", k, err)
		}
	}
}

func (s *Service) Add(ctx context.Context, userID string, r io.Reader) (Photo, error) {
	full, thumb, p, err := s.process(ctx, userID, r)
	if err != nil {
		return Photo{}, err
	}
	photo, err := s.repo.Add(ctx, userID, full, thumb, p.Width, p.Height, len(p.Full))
	if err != nil {
		s.removeFiles(ctx, full, thumb)
		return Photo{}, err
	}
	return photo, nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, r io.Reader) (Photo, error) {
	full, thumb, p, err := s.process(ctx, userID, r)
	if err != nil {
		return Photo{}, err
	}
	photo, old, err := s.repo.Replace(ctx, userID, photoID, full, thumb, p.Width, p.Height, len(p.Full))
	if err != nil {
		s.removeFiles(ctx, full, thumb)
		return Photo{}, err
	}
	s.removeFiles(ctx, old.StorageKey, old.ThumbKey)
	return photo, nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	old, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.removeFiles(ctx, old.StorageKey, old.ThumbKey)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return nil, ErrInvalidOrder
		}
		seen[id] = true
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

// Open returns the image stream when viewerID is allowed to see the photo.
func (s *Service) Open(ctx context.Context, viewerID, photoID string, thumb bool) (io.ReadSeekCloser, error) {
	p, err := s.repo.Viewable(ctx, viewerID, photoID)
	if err != nil {
		return nil, err
	}
	key := p.StorageKey
	if thumb {
		key = p.ThumbKey
	}
	f, err := s.store.Open(ctx, key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return f, err
}

// PurgeUserFiles deletes every stored file of a user (called before account deletion).
func (s *Service) PurgeUserFiles(ctx context.Context, userID string) error {
	keys, err := s.repo.KeysForUser(ctx, userID)
	if err != nil {
		return err
	}
	s.removeFiles(ctx, keys...)
	return nil
}
