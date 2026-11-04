package photos

import (
	"context"
	"errors"
	"io"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
)

type Service struct {
	repo   Repository
	store  *media.Store
	signer *media.Signer
}

func NewService(repo Repository, store *media.Store, signer *media.Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

var errNotFound = platformerrors.New(http.StatusNotFound, "not_found", "photo not found")

func (s *Service) toPhoto(p storedPhoto) Photo {
	return Photo{ID: p.ID, URL: s.signer.URL(p.ID), Position: p.Position}
}

func translateImageError(err error) error {
	switch {
	case errors.Is(err, media.ErrNotImage):
		return platformerrors.New(http.StatusUnsupportedMediaType, "unsupported_media", err.Error())
	case errors.Is(err, media.ErrTooLarge):
		return platformerrors.New(http.StatusRequestEntityTooLarge, "too_large", err.Error())
	case errors.Is(err, media.ErrTooSmall):
		return platformerrors.New(http.StatusBadRequest, "image_too_small", err.Error())
	case errors.Is(err, media.ErrUnreadable):
		return platformerrors.New(http.StatusBadRequest, "unreadable_image", err.Error())
	}
	return err
}

func (s *Service) Upload(ctx context.Context, userID string, r io.Reader) (Photo, error) {
	processed, err := media.Process(r)
	if err != nil {
		return Photo{}, translateImageError(err)
	}
	key, err := s.store.Put(processed.Data)
	if err != nil {
		return Photo{}, err
	}
	p, err := s.repo.Add(ctx, userID, key, len(processed.Data), processed.Width, processed.Height)
	if err != nil {
		_ = s.store.Delete(key)
		if errors.Is(err, ErrLimit) {
			return Photo{}, platformerrors.New(http.StatusConflict, "photo_limit", "you can have at most 6 photos")
		}
		return Photo{}, err
	}
	return s.toPhoto(p), nil
}

func (s *Service) Replace(ctx context.Context, userID, id string, r io.Reader) (Photo, error) {
	processed, err := media.Process(r)
	if err != nil {
		return Photo{}, translateImageError(err)
	}
	key, err := s.store.Put(processed.Data)
	if err != nil {
		return Photo{}, err
	}
	old, err := s.repo.Replace(ctx, userID, id, key, len(processed.Data), processed.Width, processed.Height)
	if err != nil {
		_ = s.store.Delete(key)
		if errors.Is(err, ErrNotFound) {
			return Photo{}, errNotFound
		}
		return Photo{}, err
	}
	_ = s.store.Delete(old)
	p, err := s.repo.GetOwned(ctx, userID, id)
	if err != nil {
		return Photo{}, err
	}
	return s.toPhoto(p), nil
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	key, err := s.repo.Delete(ctx, userID, id)
	if errors.Is(err, ErrNotFound) {
		return errNotFound
	}
	if err != nil {
		return err
	}
	return s.store.Delete(key)
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(rows))
	for _, p := range rows {
		out = append(out, s.toPhoto(p))
	}
	return out, nil
}

// Reorder sets the display order; the first id becomes the main photo.
func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return nil, platformerrors.New(http.StatusBadRequest, "validation", "duplicate photo id")
		}
		seen[id] = true
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, platformerrors.New(http.StatusBadRequest, "validation", "ids must list exactly your photos")
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// MakePrimary moves a photo to the first position.
func (s *Service) MakePrimary(ctx context.Context, userID, id string) ([]Photo, error) {
	rows, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := []string{id}
	found := false
	for _, p := range rows {
		if p.ID == id {
			found = true
			continue
		}
		ids = append(ids, p.ID)
	}
	if !found {
		return nil, errNotFound
	}
	return s.Reorder(ctx, userID, ids)
}

// FilePath resolves a signed URL request to a file on disk.
func (s *Service) FilePath(ctx context.Context, id, exp, sig string) (string, error) {
	if !s.signer.Verify(id, exp, sig) {
		return "", errNotFound
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", errNotFound
		}
		return "", err
	}
	path, err := s.store.Path(p.StorageKey)
	if err != nil {
		return "", errNotFound
	}
	return path, nil
}
