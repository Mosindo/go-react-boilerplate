package photos

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"

	"example.com/api/internal/platform/storage"
)

// Access decides whether a viewer may see another user's photos.
type Access interface {
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type Blobs interface {
	Put(key string, data []byte) error
	Open(key string) (io.ReadSeekCloser, error)
	Delete(key string) error
}

type Service struct {
	repo   Repository
	blobs  Blobs
	access Access
}

func NewService(repo Repository, blobs Blobs, access Access) *Service {
	return &Service{repo: repo, blobs: blobs, access: access}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

// Upload validates, cleans and stores an image. With replaceID the new image
// takes the old one's place and the old file is removed.
func (s *Service) Upload(ctx context.Context, userID string, raw []byte, replaceID string) (Photo, error) {
	processed, err := Process(raw)
	if err != nil {
		return Photo{}, err
	}
	id, err := newUUID()
	if err != nil {
		return Photo{}, err
	}
	key := id[:2] + "/" + id + ".jpg"
	if err := s.blobs.Put(key, processed.Data); err != nil {
		return Photo{}, err
	}

	photo, oldKey, err := s.repo.Add(ctx, Photo{
		UserID: userID, StorageKey: key, ContentType: "image/jpeg",
		SizeBytes: len(processed.Data), Width: processed.Width, Height: processed.Height,
	}, replaceID)
	if err != nil {
		s.deleteBlob(key) // do not leave an orphan behind
		return Photo{}, err
	}
	if oldKey != "" {
		s.deleteBlob(oldKey)
	}
	return photo, nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.deleteBlob(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}

// Open returns the photo content if the viewer is allowed to see it. Photos the
// viewer may not see answer "not found" so their existence is not disclosed.
func (s *Service) Open(ctx context.Context, viewerID, photoID string) (Photo, io.ReadSeekCloser, error) {
	photo, err := s.repo.Get(ctx, photoID)
	if err != nil {
		return Photo{}, nil, err
	}
	if photo.UserID != viewerID {
		ok, err := s.access.CanView(ctx, viewerID, photo.UserID)
		if err != nil {
			return Photo{}, nil, err
		}
		if !ok {
			return Photo{}, nil, ErrNotFound
		}
	}
	f, err := s.blobs.Open(photo.StorageKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return Photo{}, nil, ErrNotFound
		}
		return Photo{}, nil, err
	}
	return photo, f, nil
}

func (s *Service) deleteBlob(key string) {
	if err := s.blobs.Delete(key); err != nil {
		log.Printf(`{"event":"photo_blob_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
