package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"strings"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
)

var (
	errBadOrder    = platformerrors.BadRequest("invalid_order", "order must list each of your photos exactly once")
	errLastPhoto   = platformerrors.Conflict("last_photo", "you need at least one photo; add another before removing this one")
	errPhotoAbsent = platformerrors.NotFound("not_found", "photo not found")
)

// Viewers decides whether a user may see another user's photos.
type Viewers interface {
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type Service struct {
	repo    Repository
	store   storage.Storage
	viewers Viewers
}

func NewService(repo Repository, store storage.Storage, viewers Viewers) *Service {
	return &Service{repo: repo, store: store, viewers: viewers}
}

func newKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:] + ".jpg", nil
}

func thumbKey(key string) string { return strings.TrimSuffix(key, ".jpg") + "_t.jpg" }

func (s *Service) put(img processed) (string, error) {
	key, err := newKey()
	if err != nil {
		return "", err
	}
	if err := s.store.Put(key, img.Full); err != nil {
		return "", err
	}
	if err := s.store.Put(thumbKey(key), img.Thumb); err != nil {
		_ = s.store.Delete(key)
		return "", err
	}
	return key, nil
}

func (s *Service) remove(key string) {
	for _, k := range []string{key, thumbKey(key)} {
		if err := s.store.Delete(k); err != nil {
			log.Printf(`{"event":"photo_delete_failed","key":%q,"error":%q}`, k, err.Error())
		}
	}
}

func (s *Service) Upload(ctx context.Context, userID string, data []byte) ([]profiles.Photo, error) {
	img, err := processImage(data)
	if err != nil {
		return nil, err
	}
	key, err := s.put(img)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Add(ctx, userID, key, img.Width, img.Height, len(img.Full)); err != nil {
		s.remove(key)
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) ([]profiles.Photo, error) {
	img, err := processImage(data)
	if err != nil {
		return nil, err
	}
	key, err := s.put(img)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Full))
	if err != nil {
		s.remove(key)
		if errors.Is(err, ErrNotFound) {
			return nil, errPhotoAbsent
		}
		return nil, err
	}
	s.remove(oldKey)
	return s.repo.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]profiles.Photo, error) {
	list, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(list) <= 1 {
		for _, p := range list {
			if p.ID == photoID {
				return nil, errLastPhoto
			}
		}
	}
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, errPhotoAbsent
		}
		return nil, err
	}
	s.remove(key)
	return s.repo.List(ctx, userID)
}

// Reorder sets the display order; the first id becomes the main photo.
func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]profiles.Photo, error) {
	seen := map[string]bool{}
	for i, id := range ids {
		id = strings.ToLower(id)
		ids[i] = id
		if !platformerrors.IsUUID(id) || seen[id] {
			return nil, errBadOrder
		}
		seen[id] = true
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

// Open returns the image bytes for a viewer allowed to see the owner's photos.
func (s *Service) Open(ctx context.Context, viewerID, photoID string, thumb bool) (io.ReadSeekCloser, error) {
	p, err := s.repo.Get(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, errPhotoAbsent
		}
		return nil, err
	}
	ok := p.UserID == viewerID
	if !ok {
		if ok, err = s.viewers.CanView(ctx, viewerID, p.UserID); err != nil {
			return nil, err
		}
	}
	if !ok {
		return nil, errPhotoAbsent
	}
	key := p.Key
	if thumb {
		key = thumbKey(key)
	}
	f, err := s.store.Open(key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, errPhotoAbsent
		}
		return nil, err
	}
	return f, nil
}

// PrepareDelete implements auth.FilePurger: it snapshots keys before the user rows disappear.
func (s *Service) PrepareDelete(ctx context.Context, userID string) (func(), error) {
	keys, err := s.repo.Keys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() {
		for _, k := range keys {
			s.remove(k)
		}
	}, nil
}
