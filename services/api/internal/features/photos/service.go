package photos

import (
	"context"
	"errors"
	"io"
	"log"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
	"github.com/google/uuid"
)

type Service struct {
	repo  Repository
	files storage.Store
	// sem bounds concurrent image decoding (CPU and memory heavy).
	sem chan struct{}
}

func NewService(repo Repository, files storage.Store) *Service {
	return &Service{repo: repo, files: files, sem: make(chan struct{}, 2)}
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) (profiles.Photo, error) {
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return profiles.Photo{}, ctx.Err()
	}

	processed, err := Process(data)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedType):
			return profiles.Photo{}, platformerrors.FieldInvalid("file", "Only JPEG, PNG or WebP images are accepted")
		case errors.Is(err, ErrImageTooLarge):
			return profiles.Photo{}, platformerrors.FieldInvalid("file", "Image dimensions are too large")
		case errors.Is(err, ErrBadImage):
			return profiles.Photo{}, platformerrors.FieldInvalid("file", "The image is invalid or too small (min 200px)")
		}
		return profiles.Photo{}, err
	}

	id := uuid.NewString()
	fullKey := "photos/" + userID + "/" + id + ".jpg"
	thumbKey := "photos/" + userID + "/" + id + ".thumb.jpg"
	if err := s.files.Put(ctx, fullKey, processed.Full); err != nil {
		return profiles.Photo{}, err
	}
	if err := s.files.Put(ctx, thumbKey, processed.Thumb); err != nil {
		s.cleanup(fullKey)
		return profiles.Photo{}, err
	}

	stored, err := s.repo.Add(ctx, StoredPhoto{
		ID: id, UserID: userID, StorageKey: fullKey, ThumbKey: thumbKey,
		Width: processed.Width, Height: processed.Height, SizeBytes: len(processed.Full),
	})
	if err != nil {
		s.cleanup(fullKey, thumbKey)
		switch {
		case errors.Is(err, ErrNoProfile):
			return profiles.Photo{}, platformerrors.NotFound("Create your profile before adding photos")
		case errors.Is(err, ErrLimit):
			return profiles.Photo{}, platformerrors.Conflict("conflict", "You can add up to 6 photos")
		}
		return profiles.Photo{}, err
	}
	return profiles.NewPhoto(stored.ID, stored.Position), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	if !profiles.ValidUUID(photoID) {
		return platformerrors.NotFound("Photo not found")
	}
	p, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return platformerrors.NotFound("Photo not found")
		}
		return err
	}
	s.cleanup(p.StorageKey, p.ThumbKey)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]profiles.Photo, error) {
	for _, id := range ids {
		if !profiles.ValidUUID(id) {
			return nil, platformerrors.FieldInvalid("ids", "Invalid photo id")
		}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrBadOrder) {
			return nil, platformerrors.FieldInvalid("ids", "List every photo exactly once")
		}
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

type File struct {
	Content   io.ReadSeekCloser
	UpdatedAt time.Time
}

func (s *Service) Open(ctx context.Context, viewerID, photoID, size string) (File, error) {
	if !profiles.ValidUUID(photoID) {
		return File{}, platformerrors.NotFound("Photo not found")
	}
	if size != "" && size != "full" && size != "thumb" {
		return File{}, platformerrors.FieldInvalid("size", "Must be thumb or full")
	}
	fullKey, thumbKey, created, err := s.repo.FileFor(ctx, viewerID, photoID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return File{}, platformerrors.NotFound("Photo not found")
		}
		return File{}, err
	}
	key := fullKey
	if size == "thumb" {
		key = thumbKey
	}
	f, err := s.files.Open(ctx, key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return File{}, platformerrors.NotFound("Photo not found")
		}
		return File{}, err
	}
	return File{Content: f, UpdatedAt: created}, nil
}

func (s *Service) cleanup(keys ...string) {
	for _, k := range keys {
		if err := s.files.Delete(context.Background(), k); err != nil {
			log.Printf(`{"event":"photo_cleanup_failed","key":%q,"error":%q}`, k, err.Error())
		}
	}
}
