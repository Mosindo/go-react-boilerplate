package photos

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
)

var (
	ErrLimit         = platformerrors.Conflict("photo_limit", "Vous pouvez ajouter jusqu'à 6 photos.")
	ErrNotFound      = platformerrors.NotFound("photo_not_found", "Photo introuvable.")
	ErrInvalidOrder  = platformerrors.Validation("invalid_photo_order", "L'ordre doit contenir chacune de vos photos une seule fois.")
	ErrUnsupported   = platformerrors.New(415, "unsupported_image", "Format non pris en charge. Utilisez une photo JPEG, PNG ou WebP.")
	ErrTooLarge      = platformerrors.New(413, "image_too_large", "Photo trop volumineuse (10 Mo maximum).")
	ErrInvalidImage  = platformerrors.Validation("invalid_image", "Cette image est illisible ou trop petite (200 × 200 px minimum).")
	ErrMissingUpload = platformerrors.Validation("missing_photo", "Aucune photo reçue.")
)

type Service struct {
	repo    Repository
	storage media.Storage
	signer  *media.Signer
}

func NewService(repo Repository, storage media.Storage, signer *media.Signer) *Service {
	return &Service{repo: repo, storage: storage, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string) ([]profiles.PhotoView, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	views := make([]profiles.PhotoView, 0, len(stored))
	for _, p := range stored {
		views = append(views, profiles.PhotoView{ID: p.ID, URL: s.signer.URL(p.StorageKey), Width: p.Width, Height: p.Height, Position: p.Position})
	}
	return views, nil
}

// process reads at most MaxUploadBytes, sanitises the image and stores the re-encoded JPEG.
func (s *Service) process(r io.Reader) (storedPhoto, int, error) {
	data, err := io.ReadAll(io.LimitReader(r, media.MaxUploadBytes+1))
	if err != nil {
		return storedPhoto{}, 0, ErrMissingUpload
	}
	if len(data) == 0 {
		return storedPhoto{}, 0, ErrMissingUpload
	}
	if len(data) > media.MaxUploadBytes {
		return storedPhoto{}, 0, ErrTooLarge
	}
	img, err := media.Process(data)
	switch {
	case errors.Is(err, media.ErrUnsupportedType):
		return storedPhoto{}, 0, ErrUnsupported
	case errors.Is(err, media.ErrImageTooLarge):
		return storedPhoto{}, 0, ErrTooLarge
	case err != nil:
		return storedPhoto{}, 0, ErrInvalidImage
	}
	key, err := media.NewKey()
	if err != nil {
		return storedPhoto{}, 0, err
	}
	if err := s.storage.Save(key, img.Data); err != nil {
		return storedPhoto{}, 0, err
	}
	return storedPhoto{StorageKey: key, Width: img.Width, Height: img.Height}, len(img.Data), nil
}

func (s *Service) Upload(ctx context.Context, userID string, r io.Reader) ([]profiles.PhotoView, error) {
	// Fail fast before doing image work when the limit is already reached.
	existing, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= MaxPhotos {
		return nil, ErrLimit
	}
	photo, size, err := s.process(r)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Insert(ctx, userID, photo, size); err != nil {
		s.purge(photo.StorageKey)
		if errors.Is(err, errRepoLimit) {
			return nil, ErrLimit
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, r io.Reader) ([]profiles.PhotoView, error) {
	photo, size, err := s.process(r)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, photo, size)
	if err != nil {
		s.purge(photo.StorageKey)
		if errors.Is(err, errRepoNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.purge(oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]profiles.PhotoView, error) {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, errRepoNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.purge(key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]profiles.PhotoView, error) {
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(id)))
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		if errors.Is(err, errRepoMismatch) {
			return nil, ErrInvalidOrder
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// OpenMedia verifies a signed URL and opens the stored file.
func (s *Service) OpenMedia(key, exp, sig string) (io.ReadSeekCloser, time.Time, bool) {
	if !media.ValidKey(key) || !s.signer.Verify(key, exp, sig) {
		return nil, time.Time{}, false
	}
	f, mod, err := s.storage.Open(key)
	if err != nil {
		return nil, time.Time{}, false
	}
	return f, mod, true
}

func (s *Service) purge(key string) {
	if err := s.storage.Delete(key); err != nil {
		log.Printf(`{"event":"photo_purge_failed","key":%q,"error":%q}`, key, err.Error())
	}
}
