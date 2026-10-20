package photos

import (
	"context"
	"errors"
	"log"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
	"github.com/google/uuid"
)

// Visibility decides whether a viewer may see a user's photos.
type Visibility interface {
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type Service struct {
	repo       Repository
	store      storage.Store
	visibility Visibility
}

func NewService(repo Repository, store storage.Store, visibility Visibility) *Service {
	return &Service{repo: repo, store: store, visibility: visibility}
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (Photo, error) {
	processed, w, h, err := ProcessImage(raw)
	if err != nil {
		return Photo{}, err
	}
	key := uuid.NewString() + ".jpg"
	if err := s.store.Save(key, processed); err != nil {
		return Photo{}, err
	}
	photo, err := s.repo.Add(ctx, userID, NewPhoto{StorageKey: key, Width: w, Height: h, SizeBytes: len(processed)})
	if err != nil {
		if delErr := s.store.Delete(key); delErr != nil {
			log.Printf("cleanup orphan photo %s: %v", key, delErr)
		}
		return Photo{}, err
	}
	return photo, nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return nil, platformerrors.BadRequest("invalid photo id")
		}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	if _, err := uuid.Parse(photoID); err != nil {
		return errNotFound
	}
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	if err := s.store.Delete(key); err != nil {
		log.Printf("delete photo file %s: %v", key, err)
	}
	return nil
}

// Open returns the image bytes if the viewer may see them. Unknown, hidden or
// blocked photos are all reported as "not found".
func (s *Service) Open(ctx context.Context, viewerID, photoID string) ([]byte, error) {
	if _, err := uuid.Parse(photoID); err != nil {
		return nil, errNotFound
	}
	photo, err := s.repo.Get(ctx, photoID)
	if err != nil {
		return nil, err
	}
	ok, err := s.visibility.CanView(ctx, viewerID, photo.OwnerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errNotFound
	}
	data, err := s.store.Read(photo.StorageKey)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, errNotFound
	}
	return data, err
}
