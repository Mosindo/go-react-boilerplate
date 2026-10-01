package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/storage"
)

var ErrForbidden = errors.New("photo not accessible")

// Access decides whether a viewer may see another user's photos
// (implemented by profiles.Service).
type Access interface {
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type Service struct {
	repo   Repository
	store  storage.Store
	access Access
}

func NewService(repo Repository, store storage.Store, access Access) *Service {
	return &Service{repo: repo, store: store, access: access}
}

func newKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ".jpg", nil
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toPhoto())
	}
	return out, nil
}

func (s *Service) prepare(raw []byte) (processed, string, error) {
	img, err := processImage(raw)
	if err != nil {
		if errors.Is(err, ErrUnsupportedType) || errors.Is(err, ErrImageTooLarge) ||
			errors.Is(err, ErrImageTooSmall) || errors.Is(err, ErrInvalidImage) {
			return processed{}, "", httpx.Invalid(err.Error())
		}
		return processed{}, "", err
	}
	key, err := newKey()
	if err != nil {
		return processed{}, "", err
	}
	if err := s.store.Save(key, img.Data); err != nil {
		return processed{}, "", err
	}
	return img, key, nil
}

func (s *Service) Add(ctx context.Context, userID string, raw []byte) (Photo, error) {
	img, key, err := s.prepare(raw)
	if err != nil {
		return Photo{}, err
	}
	row, err := s.repo.Add(ctx, userID, key, len(img.Data), img.Width, img.Height)
	if err != nil {
		s.deleteQuietly(key)
		return Photo{}, err
	}
	return row.toPhoto(), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (Photo, error) {
	img, key, err := s.prepare(raw)
	if err != nil {
		return Photo{}, err
	}
	row, oldKey, err := s.repo.Replace(ctx, userID, photoID, key, len(img.Data), img.Width, img.Height)
	if err != nil {
		s.deleteQuietly(key)
		return Photo{}, err
	}
	s.deleteQuietly(oldKey)
	return row.toPhoto(), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.deleteQuietly(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return nil, httpx.Invalid(ErrBadOrder.Error())
		}
		seen[id] = struct{}{}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrBadOrder) {
			return nil, httpx.Invalid(err.Error())
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open returns the photo bytes if viewerID is allowed to see them.
func (s *Service) Open(ctx context.Context, viewerID, photoID string) (io.ReadCloser, error) {
	row, err := s.repo.Get(ctx, photoID)
	if err != nil {
		return nil, err
	}
	if row.UserID != viewerID {
		ok, err := s.access.CanView(ctx, viewerID, row.UserID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrPhotoNotFound // do not reveal existence
		}
	}
	rc, err := s.store.Open(row.StorageKey)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrPhotoNotFound
	}
	return rc, err
}

func (s *Service) deleteQuietly(key string) {
	if err := s.store.Delete(key); err != nil {
		log.Printf(`{"event":"photo_cleanup_failed","error":%q}`, err.Error())
	}
}
