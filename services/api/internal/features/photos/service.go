package photos

import (
	"context"
	"errors"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
)

var (
	errNotFound = platformerrors.NotFound("photo_not_found", "photo not found")
	errLimit    = platformerrors.Conflict("photo_limit", "you can upload at most 6 photos")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) ListByUsers(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	return s.repo.ListByUsers(ctx, userIDs)
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) (Photo, error) {
	processed, err := Process(data)
	if err != nil {
		return Photo{}, err
	}
	photo, err := s.repo.Add(ctx, userID, processed)
	if errors.Is(err, ErrRepositoryLimit) {
		return Photo{}, errLimit
	}
	return photo, err
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) (Photo, error) {
	processed, err := Process(data)
	if err != nil {
		return Photo{}, err
	}
	photo, err := s.repo.Replace(ctx, userID, photoID, processed)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Photo{}, errNotFound
	}
	return photo, err
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	err := s.repo.Delete(ctx, userID, photoID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return errNotFound
	}
	return err
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	seen := make(map[string]struct{}, len(ids))
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(id)
		if !httpx.IsUUID(id) {
			return nil, platformerrors.BadRequest("invalid_request", "photoIds must be UUIDs")
		}
		if _, dup := seen[id]; dup {
			return nil, platformerrors.BadRequest("invalid_request", "photoIds contains duplicates")
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		if errors.Is(err, ErrRepositoryMismatch) {
			return nil, platformerrors.BadRequest("invalid_order", "photoIds must list every one of your photos exactly once")
		}
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

func (s *Service) Blob(ctx context.Context, viewerID, photoID string, thumb bool) (Blob, error) {
	b, err := s.repo.GetBlob(ctx, viewerID, photoID, thumb)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Blob{}, errNotFound
	}
	return b, err
}
