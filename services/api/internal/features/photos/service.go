package photos

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/google/uuid"
)

type Service struct {
	repo    *Repository
	storage Storage
}

func NewService(repo *Repository, storage Storage) *Service {
	return &Service{repo: repo, storage: storage}
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, ErrLimit):
		return platformerrors.Conflict("photo_limit", "you can upload up to 6 photos")
	case errors.Is(err, ErrNotFound):
		return platformerrors.ErrNotFound
	case errors.Is(err, ErrOrder):
		return platformerrors.Invalid("ids must list every photo exactly once")
	}
	return err
}

func (s *Service) store(userID string, src io.Reader) (Processed, string, error) {
	processed, err := Process(src)
	if err != nil {
		return Processed{}, "", err
	}
	key := userID + "/" + uuid.NewString() + ".jpg"
	if err := s.storage.Save(key, processed.Data); err != nil {
		return Processed{}, "", err
	}
	return processed, key, nil
}

func (s *Service) Upload(ctx context.Context, userID string, src io.Reader) (Photo, error) {
	processed, key, err := s.store(userID, src)
	if err != nil {
		return Photo{}, err
	}
	photo, err := s.repo.Add(ctx, userID, key, processed.Width, processed.Height, len(processed.Data))
	if err != nil {
		_ = s.storage.Delete(key)
		return Photo{}, mapErr(err)
	}
	return photo, nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, src io.Reader) (Photo, error) {
	processed, key, err := s.store(userID, src)
	if err != nil {
		return Photo{}, err
	}
	photo, oldKey, err := s.repo.Replace(ctx, userID, photoID, key, processed.Width, processed.Height, len(processed.Data))
	if err != nil {
		_ = s.storage.Delete(key)
		return Photo{}, mapErr(err)
	}
	_ = s.storage.Delete(oldKey)
	return photo, nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return mapErr(err)
	}
	_ = s.storage.Delete(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return platformerrors.Invalid("invalid photo id")
		}
	}
	return mapErr(s.repo.Reorder(ctx, userID, ids))
}

func (s *Service) Open(ctx context.Context, viewerID, photoID string) (*os.File, error) {
	key, err := s.repo.KeyForViewer(ctx, viewerID, photoID)
	if err != nil {
		return nil, mapErr(err)
	}
	f, err := s.storage.Open(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil, platformerrors.ErrNotFound
	}
	return f, err
}

func IsMultipartTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}
