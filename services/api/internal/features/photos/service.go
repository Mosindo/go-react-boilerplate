package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
)

var (
	ErrPhotoNotFound = platformerrors.ErrNotFound
	ErrTooMany       = platformerrors.Conflict("maximum 6 photos")
)

type Viewer interface {
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type Service struct {
	repo    Repository
	store   storage.Storage
	viewers Viewer
}

func NewService(repo Repository, store storage.Storage, viewers Viewer) *Service {
	return &Service{repo: repo, store: store, viewers: viewers}
}

func newKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ".jpg", nil
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (Photo, error) {
	data, w, h, err := processImage(raw)
	if err != nil {
		return Photo{}, err
	}
	key, err := newKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.store.Put(ctx, key, data); err != nil {
		return Photo{}, err
	}
	rec, err := s.repo.Add(ctx, userID, key, w, h, len(data), profiles.MaxPhotos)
	if err != nil {
		_ = s.store.Delete(ctx, key)
		if errors.Is(err, ErrRepositoryFull) {
			return Photo{}, ErrTooMany
		}
		return Photo{}, err
	}
	return s.toPhoto(ctx, rec.ID, userID)
}

func (s *Service) Replace(ctx context.Context, userID, id string, raw []byte) (Photo, error) {
	if !profiles.IsUUID(id) {
		return Photo{}, ErrPhotoNotFound
	}
	data, w, h, err := processImage(raw)
	if err != nil {
		return Photo{}, err
	}
	key, err := newKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.store.Put(ctx, key, data); err != nil {
		return Photo{}, err
	}
	old, err := s.repo.Replace(ctx, userID, id, key, w, h, len(data))
	if err != nil {
		_ = s.store.Delete(ctx, key)
		if errors.Is(err, ErrRepositoryNotFound) {
			return Photo{}, ErrPhotoNotFound
		}
		return Photo{}, err
	}
	s.deleteFile(ctx, old)
	return s.toPhoto(ctx, id, userID)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if !profiles.IsUUID(id) {
		return ErrPhotoNotFound
	}
	key, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrPhotoNotFound
		}
		return err
	}
	s.deleteFile(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	seen := map[string]bool{}
	for _, id := range ids {
		if !profiles.IsUUID(id) || seen[id] {
			return nil, platformerrors.Invalid("ids must be a list of distinct photo ids")
		}
		seen[id] = true
	}
	if len(ids) == 0 {
		return nil, platformerrors.Invalid("ids must not be empty")
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrRepositoryMismatch) {
			return nil, platformerrors.Invalid("ids must list all of your photos exactly once")
		}
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

// Open returns the file of a photo after checking that viewerID is allowed to see its owner.
func (s *Service) Open(ctx context.Context, viewerID, id string) (io.ReadSeekCloser, error) {
	if !profiles.IsUUID(id) {
		return nil, ErrPhotoNotFound
	}
	rec, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	ok, err := s.viewers.CanView(ctx, viewerID, rec.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPhotoNotFound
	}
	f, err := s.store.Open(ctx, rec.StorageKey)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrPhotoNotFound
	}
	return f, err
}

func (s *Service) toPhoto(ctx context.Context, id, userID string) (Photo, error) {
	all, err := s.repo.List(ctx, userID)
	if err != nil {
		return Photo{}, err
	}
	for _, p := range all {
		if p.ID == id {
			return p, nil
		}
	}
	return Photo{}, ErrPhotoNotFound
}

func (s *Service) deleteFile(ctx context.Context, key string) {
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"file_delete_error","key":%q,"error":%q}`, key, err.Error())
	}
}
