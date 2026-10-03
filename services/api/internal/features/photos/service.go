package photos

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
)

type Service struct {
	repo    Repository
	storage Storage
}

func NewService(repo Repository, storage Storage) *Service {
	return &Service{repo: repo, storage: storage}
}

func newStorageKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x.jpg", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// Upload validates, normalizes and stores an image. With replaceID the new picture takes
// over the position of an existing one.
func (s *Service) Upload(ctx context.Context, userID, replaceID string, file io.Reader) (Photo, error) {
	raw, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		return Photo{}, err
	}
	if len(raw) > maxUploadBytes {
		return Photo{}, ErrTooBig
	}
	img, err := processImage(raw)
	if err != nil {
		return Photo{}, err
	}
	key, err := newStorageKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.storage.Put(ctx, key, img.data); err != nil {
		return Photo{}, err
	}
	photo, oldKey, err := s.repo.Add(ctx, NewPhoto{
		UserID: userID, StorageKey: key, Size: len(img.data), Width: img.width, Height: img.height, ReplaceID: replaceID,
	})
	if err != nil {
		s.deleteQuietly(ctx, key)
		return Photo{}, err
	}
	if oldKey != "" {
		s.deleteQuietly(ctx, oldKey)
	}
	return photo, nil
}

var ErrTooBig = errors.New("image is larger than 8 MB")

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.deleteQuietly(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}

func (s *Service) Open(ctx context.Context, viewerID, photoID string) (io.ReadCloser, error) {
	key, err := s.repo.StorageKeyForViewer(ctx, viewerID, photoID)
	if err != nil {
		return nil, err
	}
	rc, err := s.storage.Open(ctx, key)
	if err != nil {
		return nil, ErrNotFound
	}
	return rc, nil
}

// PurgeUserFiles removes every stored file of a user (called before account deletion).
func (s *Service) PurgeUserFiles(ctx context.Context, userID string) error {
	keys, err := s.repo.StorageKeysForUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := s.storage.Delete(ctx, k); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) deleteQuietly(ctx context.Context, key string) {
	if err := s.storage.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"photo_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}
