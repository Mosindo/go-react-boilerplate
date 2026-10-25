package photos

import (
	"context"

	"example.com/api/internal/features/profiles"
)

// Viewer decides whether a user may see another user's profile (and therefore their photos).
type Viewer interface {
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type Service struct {
	repo   *Repository
	viewer Viewer
}

func NewService(repo *Repository, viewer Viewer) *Service {
	return &Service{repo: repo, viewer: viewer}
}

func (s *Service) Add(ctx context.Context, userID string, data []byte) (profiles.Photo, error) {
	p, err := process(data)
	if err != nil {
		return profiles.Photo{}, err
	}
	return s.repo.add(ctx, userID, p)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) (profiles.Photo, error) {
	p, err := process(data)
	if err != nil {
		return profiles.Photo{}, err
	}
	return s.repo.replace(ctx, userID, photoID, p)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	return s.repo.remove(ctx, userID, photoID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]profiles.Photo, error) {
	return s.repo.reorder(ctx, userID, ids)
}

// Image returns the bytes of a photo if the viewer may see its owner. Unauthorized access is reported as ErrNotFound.
func (s *Service) Image(ctx context.Context, viewerID, photoID string, thumb bool) ([]byte, error) {
	owner, err := s.repo.ownerOf(ctx, photoID)
	if err != nil {
		return nil, err
	}
	if owner != viewerID {
		ok, err := s.viewer.CanView(ctx, viewerID, owner)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
	}
	return s.repo.content(ctx, photoID, thumb)
}
