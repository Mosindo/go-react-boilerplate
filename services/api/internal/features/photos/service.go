package photos

import (
	"context"
	"errors"
	"io"
	"log"
	"runtime"
	"strings"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
)

const MaxPhotos = 6

var (
	ErrPhotoNotFound = apperr.NotFound("photo not found")
	ErrTooManyPhotos = apperr.Conflict("photo_limit", "you can upload up to 6 photos")
	ErrOrderMismatch = apperr.Validation("photoIds must list every one of your photos exactly once")
)

type Service struct {
	repo   Repository
	store  storage.Store
	signer *media.Signer
	// processing bounds concurrent image decodes (CPU and memory heavy).
	processing chan struct{}
}

func NewService(repo Repository, store storage.Store, signer *media.Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer, processing: make(chan struct{}, runtime.NumCPU())}
}

func (s *Service) process(ctx context.Context, raw []byte) (ProcessedImage, error) {
	select {
	case s.processing <- struct{}{}:
	case <-ctx.Done():
		return ProcessedImage{}, ctx.Err()
	}
	defer func() { <-s.processing }()
	return ProcessImage(raw)
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	ids, err := s.repo.ListIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	photos := make([]Photo, 0, len(ids))
	for _, id := range ids {
		photos = append(photos, Photo{ID: id, URL: s.signer.PhotoURL(id)})
	}
	return photos, nil
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) ([]Photo, error) {
	img, err := s.process(ctx, raw)
	if err != nil {
		return nil, err
	}
	key, err := s.save(ctx, userID, img)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Insert(ctx, userID, key, img.Width, img.Height, len(img.Data), MaxPhotos); err != nil {
		s.deleteQuietly(ctx, key)
		if errors.Is(err, ErrRepositoryFull) {
			return nil, ErrTooManyPhotos
		}
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, apperr.NotFound("profile not found")
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) ([]Photo, error) {
	img, err := s.process(ctx, raw)
	if err != nil {
		return nil, err
	}
	key, err := s.save(ctx, userID, img)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.deleteQuietly(ctx, key)
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	s.deleteQuietly(ctx, oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]Photo, error) {
	oldKey, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	s.deleteQuietly(ctx, oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(id)
		if seen[id] {
			return nil, ErrOrderMismatch
		}
		seen[id] = true
		normalized = append(normalized, id)
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		if errors.Is(err, ErrRepositoryMismatch) {
			return nil, ErrOrderMismatch
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open returns the file for a signed media URL.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (io.ReadSeekCloser, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return nil, apperr.Forbidden("invalid or expired link")
	}
	key, err := s.repo.StorageKey(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	file, err := s.store.Open(ctx, key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	return file, nil
}

// PrepareAccountDeletion implements auth.AccountCleaner: files are removed
// only after the database deletion succeeded.
func (s *Service) PrepareAccountDeletion(ctx context.Context, userID string) (func(context.Context), error) {
	keys, err := s.repo.ListKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) {
		for _, key := range keys {
			s.deleteQuietly(ctx, key)
		}
	}, nil
}

func (s *Service) save(ctx context.Context, userID string, img ProcessedImage) (string, error) {
	key, err := storage.NewKey("photos/"+userID, ".jpg")
	if err != nil {
		return "", err
	}
	if err := s.store.Put(ctx, key, img.Data); err != nil {
		return "", err
	}
	return key, nil
}

func (s *Service) deleteQuietly(ctx context.Context, key string) {
	if err := s.store.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"photo_file_delete_failed","error":%q}`, err.Error())
	}
}
