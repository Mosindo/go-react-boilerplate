package photos

import (
	"context"
	"errors"
	"io"
)

// Viewer decides whether a user may see another user's media.
type Viewer interface {
	CanView(ctx context.Context, viewerID, ownerID string) (bool, error)
}

type Service struct {
	repo   Repository
	viewer Viewer
}

func NewService(repo Repository, viewer Viewer) *Service {
	return &Service{repo: repo, viewer: viewer}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Upload(ctx context.Context, userID string, src io.Reader) (Photo, error) {
	p, err := Process(src)
	if err != nil {
		return Photo{}, err
	}
	return s.repo.Add(ctx, userID, p)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, src io.Reader) (Photo, error) {
	p, err := Process(src)
	if err != nil {
		return Photo{}, err
	}
	return s.repo.Replace(ctx, userID, photoID, p)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	return s.repo.Delete(ctx, userID, photoID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return ErrBadReorder
		}
		seen[id] = true
	}
	return s.repo.Reorder(ctx, userID, ids)
}

// Image returns the bytes only if viewerID is allowed to see the owner's profile.
// Unauthorized and missing photos are indistinguishable (ErrNotFound).
func (s *Service) Image(ctx context.Context, viewerID, photoID string, thumbnail bool) (StoredImage, error) {
	img, err := s.repo.Image(ctx, photoID, thumbnail)
	if err != nil {
		return StoredImage{}, err
	}
	ok, err := s.viewer.CanView(ctx, viewerID, img.OwnerID)
	if err != nil {
		return StoredImage{}, err
	}
	if !ok {
		return StoredImage{}, ErrNotFound
	}
	return img, nil
}

func IsUploadError(err error) bool {
	return errors.Is(err, ErrUnsupportedType) || errors.Is(err, ErrTooLarge) ||
		errors.Is(err, ErrTooSmall) || errors.Is(err, ErrCorrupt)
}
