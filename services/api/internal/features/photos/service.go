package photos

import (
	"context"
	"time"

	"example.com/api/internal/platform/signedurl"
)

type Service struct {
	repo   *Repository
	signer *signedurl.Signer
}

func NewService(repo *Repository, signer *signedurl.Signer) *Service {
	return &Service{repo: repo, signer: signer}
}

func (s *Service) toPhotos(rows []row) []Photo {
	now := time.Now()
	out := make([]Photo, 0, len(rows))
	for i, r := range rows {
		out = append(out, Photo{ID: r.ID, Position: r.Position, IsPrimary: i == 0, URL: s.signer.PhotoURL(r.ID, now)})
	}
	return out
}

func (s *Service) single(r row) Photo {
	return Photo{ID: r.ID, Position: r.Position, IsPrimary: r.Position == 0, URL: s.signer.PhotoURL(r.ID, time.Now())}
}

func (s *Service) Add(ctx context.Context, userID string, raw []byte) (Photo, error) {
	data, err := ProcessImage(raw)
	if err != nil {
		return Photo{}, err
	}
	r, err := s.repo.Add(ctx, userID, data)
	if err != nil {
		return Photo{}, err
	}
	return s.single(r), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (Photo, error) {
	data, err := ProcessImage(raw)
	if err != nil {
		return Photo{}, err
	}
	r, err := s.repo.Replace(ctx, userID, photoID, data)
	if err != nil {
		return Photo{}, err
	}
	return s.single(r), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	return s.repo.Delete(ctx, userID, photoID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	rows, err := s.repo.Reorder(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	return s.toPhotos(rows), nil
}

// Content returns image bytes if the signed URL parameters are valid.
func (s *Service) Content(ctx context.Context, id, exp, sig string) ([]byte, string, error) {
	if !s.signer.Verify(id, exp, sig, time.Now()) {
		return nil, "", ErrNotFound
	}
	return s.repo.Content(ctx, id)
}
