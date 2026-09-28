package photos

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/imaging"
)

type Service struct {
	repo Repository
	// sem bounds concurrent decode/encode work (each can hold tens of MB).
	sem chan struct{}
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, sem: make(chan struct{}, 4)}
}

func (s *Service) process(ctx context.Context, data []byte) (imaging.Result, error) {
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return imaging.Result{}, ctx.Err()
	}
	res, err := imaging.Process(data)
	switch {
	case err == nil:
		return res, nil
	case errors.Is(err, imaging.ErrTooLarge):
		return imaging.Result{}, httpx.PayloadTooLarge("image dimensions are too large")
	case errors.Is(err, imaging.ErrUnsupported), errors.Is(err, imaging.ErrCorrupt):
		return imaging.Result{}, httpx.UnsupportedMedia("only valid JPEG or PNG images are accepted")
	default:
		return imaging.Result{}, err
	}
}

func (s *Service) Add(ctx context.Context, userID string, data []byte) (profiles.Photo, error) {
	img, err := s.process(ctx, data)
	if err != nil {
		return profiles.Photo{}, err
	}
	p, err := s.repo.Add(ctx, userID, img)
	if errors.Is(err, ErrLimit) {
		return profiles.Photo{}, httpx.Conflict("you can upload at most 6 photos")
	}
	return p, err
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) (profiles.Photo, error) {
	if !httpx.IsUUID(photoID) {
		return profiles.Photo{}, httpx.NotFound("photo not found")
	}
	img, err := s.process(ctx, data)
	if err != nil {
		return profiles.Photo{}, err
	}
	p, err := s.repo.Replace(ctx, userID, photoID, img)
	if errors.Is(err, ErrNotFound) {
		return profiles.Photo{}, httpx.NotFound("photo not found")
	}
	return p, err
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	if !httpx.IsUUID(photoID) {
		return httpx.NotFound("photo not found")
	}
	if err := s.repo.Delete(ctx, userID, photoID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return httpx.NotFound("photo not found")
		}
		return err
	}
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]profiles.Photo, error) {
	norm := make([]string, len(ids))
	for i, id := range ids {
		n, ok := httpx.NormalizeUUID(id)
		if !ok {
			return nil, httpx.BadRequest("photoIds must be photo ids")
		}
		norm[i] = n
	}
	list, err := s.repo.Reorder(ctx, userID, norm)
	if errors.Is(err, ErrNotPerm) {
		return nil, httpx.BadRequest("photoIds must be a permutation of your photos")
	}
	if list == nil {
		list = []profiles.Photo{}
	}
	return list, err
}

func (s *Service) Content(ctx context.Context, viewerID, photoID string) ([]byte, error) {
	id, ok := httpx.NormalizeUUID(photoID)
	if !ok {
		return nil, httpx.NotFound("photo not found")
	}
	data, err := s.repo.Content(ctx, viewerID, id)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NotFound("photo not found")
	}
	return data, err
}
