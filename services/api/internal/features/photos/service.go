package photos

import (
	"context"
	"errors"
	"io"
	"log"
	"os"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
	"github.com/google/uuid"
)

var (
	errNoProfile = platformerrors.NotFound("profile_not_found", "create your profile before adding photos")
	errNotFound  = platformerrors.NotFound("not_found", "photo not found")
)

// Blobs is the file storage contract (satisfied by storage.Local).
type Blobs interface {
	Put(key string, data []byte) error
	Open(key string) (io.ReadSeekCloser, os.FileInfo, error)
	Remove(key string) error
}

// Visibility answers whether a viewer may see a user's content (profiles.Service).
type Visibility interface {
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type Service struct {
	repo       Repository
	blobs      Blobs
	visibility Visibility
}

func NewService(repo Repository, blobs Blobs, visibility Visibility) *Service {
	return &Service{repo: repo, blobs: blobs, visibility: visibility}
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (Photo, error) {
	img, err := processImage(raw)
	switch {
	case errors.Is(err, errUnsupportedType):
		return Photo{}, platformerrors.New(415, "unsupported_media_type", "only JPEG, PNG and WebP images are accepted")
	case errors.Is(err, errBadImage):
		return Photo{}, platformerrors.BadRequest("invalid_image", "the image is invalid or its dimensions are out of range (min 200px, max 12000px)")
	case err != nil:
		return Photo{}, err
	}

	id := uuid.NewString()
	key := storage.Key(id)
	if err := s.blobs.Put(key, img.Data); err != nil {
		return Photo{}, err
	}
	photo, err := s.repo.Add(ctx, userID, id, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		_ = s.blobs.Remove(key)
		switch {
		case errors.Is(err, ErrRepoNoProfile):
			return Photo{}, errNoProfile
		case errors.Is(err, ErrRepoLimit):
			return Photo{}, platformerrors.Conflict("photo_limit", "you can have at most 6 photos")
		}
		return Photo{}, err
	}
	return photo, nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil || seen[id] {
			return nil, platformerrors.BadRequest("invalid_order", "photo order is invalid")
		}
		seen[id] = true
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		switch {
		case errors.Is(err, ErrRepoNoProfile):
			return nil, errNoProfile
		case errors.Is(err, ErrRepoBadOrdering):
			return nil, platformerrors.BadRequest("invalid_order", "photo order must list each of your photos exactly once")
		}
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return errNotFound
		}
		return err
	}
	if err := s.blobs.Remove(key); err != nil {
		log.Printf(`{"event":"file_cleanup_error","operation":"photos.delete","error":%q}`, err.Error())
	}
	return nil
}

// Open returns the file if the viewer is allowed to see its owner; otherwise "not found".
func (s *Service) Open(ctx context.Context, viewerID, photoID string) (io.ReadSeekCloser, os.FileInfo, error) {
	p, err := s.repo.Get(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return nil, nil, errNotFound
		}
		return nil, nil, err
	}
	ok, err := s.visibility.CanView(ctx, viewerID, p.UserID)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, errNotFound
	}
	f, info, err := s.blobs.Open(p.StorageKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, errNotFound
		}
		return nil, nil, err
	}
	return f, info, nil
}
