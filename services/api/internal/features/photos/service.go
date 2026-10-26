package photos

import (
	"context"
	"errors"
	"io"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/signing"
)

var (
	ErrNotFound   = errors.New("photo not found")
	ErrLimit      = errors.New("you can upload up to 6 photos")
	ErrBadOrder   = errors.New("the order must list each of your photos exactly once")
	ErrFileTooBig = errors.New("the file is too large (8 MB maximum)")
)

type Service struct {
	repo   Repository
	signer *signing.Signer
}

func NewService(repo Repository, signer *signing.Signer) *Service {
	return &Service{repo: repo, signer: signer}
}

func (s *Service) list(ctx context.Context, userID string) (ListResponse, error) {
	ids, err := s.repo.List(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	out := ListResponse{Photos: make([]profiles.Photo, 0, len(ids))}
	for _, id := range ids {
		out.Photos = append(out.Photos, profiles.Photo{ID: id, URL: s.signer.PhotoURL(id)})
	}
	return out, nil
}

func (s *Service) List(ctx context.Context, userID string) (ListResponse, error) {
	return s.list(ctx, userID)
}

func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxUploadBytes {
		return nil, ErrFileTooBig
	}
	return data, nil
}

func (s *Service) Add(ctx context.Context, userID string, file io.Reader) (ListResponse, error) {
	raw, err := readLimited(file)
	if err != nil {
		return ListResponse{}, err
	}
	img, err := processImage(raw)
	if err != nil {
		return ListResponse{}, err
	}
	if _, err := s.repo.Add(ctx, userID, img); err != nil {
		if errors.Is(err, ErrRepositoryLimit) {
			return ListResponse{}, ErrLimit
		}
		return ListResponse{}, err
	}
	return s.list(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, file io.Reader) (ListResponse, error) {
	raw, err := readLimited(file)
	if err != nil {
		return ListResponse{}, err
	}
	img, err := processImage(raw)
	if err != nil {
		return ListResponse{}, err
	}
	if _, err := s.repo.Replace(ctx, userID, photoID, img); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ListResponse{}, ErrNotFound
		}
		return ListResponse{}, err
	}
	return s.list(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) (ListResponse, error) {
	if err := s.repo.Delete(ctx, userID, photoID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ListResponse{}, ErrNotFound
		}
		return ListResponse{}, err
	}
	return s.list(ctx, userID)
}

// Reorder sets the display order; the first id becomes the main photo.
func (s *Service) Reorder(ctx context.Context, userID string, ids []string) (ListResponse, error) {
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrRepositoryOrder) {
			return ListResponse{}, ErrBadOrder
		}
		return ListResponse{}, err
	}
	return s.list(ctx, userID)
}

// Open returns the file for a signed URL. The signature proves the URL was
// issued by this API to a viewer allowed to see the photo.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (StoredFile, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return StoredFile{}, ErrNotFound
	}
	f, err := s.repo.GetFile(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return StoredFile{}, ErrNotFound
		}
		return StoredFile{}, err
	}
	return f, nil
}
