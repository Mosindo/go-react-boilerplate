package photos

import (
	"context"
	"io"
)

// Visibility decides whether a viewer may see another user's profile media.
type Visibility interface {
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type Service struct {
	repo Repository
	vis  Visibility
	sem  chan struct{}
}

func NewService(repo Repository, vis Visibility) *Service {
	return &Service{repo: repo, vis: vis, sem: make(chan struct{}, maxConcurrentIO)}
}

// processBounded limits concurrent image decoding to cap memory use.
func (s *Service) processBounded(ctx context.Context, r io.Reader) ([]byte, error) {
	raw, err := ReadLimited(r, MaxUploadBytes)
	if err != nil {
		return nil, err
	}
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return Process(raw)
}

func (s *Service) List(ctx context.Context, userID string) ([]PhotoResponse, error) {
	ids, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toResponses(ids), nil
}

func (s *Service) Add(ctx context.Context, userID string, r io.Reader) ([]PhotoResponse, error) {
	data, err := s.processBounded(ctx, r)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Add(ctx, userID, data); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, r io.Reader) ([]PhotoResponse, error) {
	data, err := s.processBounded(ctx, r)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Replace(ctx, userID, photoID, data); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]PhotoResponse, error) {
	if err := s.repo.Delete(ctx, userID, photoID); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]PhotoResponse, error) {
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Fetch returns the JPEG bytes when viewer owns the photo or may see its owner's profile.
func (s *Service) Fetch(ctx context.Context, viewerID, photoID string) ([]byte, error) {
	owner, data, err := s.repo.Get(ctx, photoID)
	if err != nil {
		return nil, err
	}
	if owner != viewerID {
		ok, err := s.vis.CanView(ctx, viewerID, owner)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
	}
	return data, nil
}

func toResponses(ids []string) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(ids))
	for _, id := range ids {
		out = append(out, PhotoResponse{ID: id, URL: "/photos/" + id})
	}
	return out
}
