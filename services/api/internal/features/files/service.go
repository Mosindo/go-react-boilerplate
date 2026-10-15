package files

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/features/users"
	"example.com/api/internal/platform/blob"
	platformerrors "example.com/api/internal/platform/errors"
)

var (
	ErrPhotoNotFound   = platformerrors.NotFound("photo_not_found", "photo not found")
	ErrPhotoLimit      = platformerrors.Conflict("photo_limit", "you can add up to 6 photos")
	ErrUnsupportedType = platformerrors.New(415, "unsupported_image", "only JPEG, PNG or WebP images are accepted")
	ErrImageTooLarge   = platformerrors.New(413, "image_too_large", "the image is too large")
	ErrImageTooSmall   = platformerrors.BadRequest("image_too_small", "the image must be at least 200 pixels wide and tall")
	ErrBadReorder      = platformerrors.BadRequest("invalid_order", "the order must list each of your photos exactly once")
	ErrBadSignature    = platformerrors.NotFound("photo_not_found", "photo not found")
)

// urlWindow is the granularity of signed-URL expiry. URLs stay identical
// during a window so image caches keep working, and live between one and two windows.
const urlWindow = 24 * time.Hour

type Service struct {
	repo   Repository
	store  blob.Store
	urlKey []byte
	now    func() time.Time
}

// NewService builds the photo service. urlSecret is the API secret; a
// purpose-bound key is derived from it so it is never used for two things.
func NewService(repo Repository, store blob.Store, urlSecret []byte) *Service {
	mac := hmac.New(sha256.New, urlSecret)
	mac.Write([]byte("photo-url-v1"))
	return &Service{repo: repo, store: store, urlKey: mac.Sum(nil), now: time.Now}
}

// ---- reading ----

// PhotosForUsers implements users.PhotoSource with a single query.
func (s *Service) PhotosForUsers(ctx context.Context, userIDs []string) (map[string][]users.PhotoView, error) {
	photos, err := s.repo.ListByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]users.PhotoView, len(userIDs))
	for _, p := range photos {
		out[p.UserID] = append(out[p.UserID], s.view(p))
	}
	return out, nil
}

func (s *Service) List(ctx context.Context, userID string) ([]users.PhotoView, error) {
	m, err := s.PhotosForUsers(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	if m[userID] == nil {
		return []users.PhotoView{}, nil
	}
	return m[userID], nil
}

func (s *Service) view(p Photo) users.PhotoView {
	exp := s.expiry()
	return users.PhotoView{
		ID:       p.ID,
		URL:      s.url(p.ID, "full", exp),
		ThumbURL: s.url(p.ID, "thumb", exp),
		Position: p.Position,
		Width:    p.Width,
		Height:   p.Height,
	}
}

// ---- writing ----

func (s *Service) Upload(ctx context.Context, userID string, data []byte) ([]users.PhotoView, error) {
	img, err := processImage(data)
	if err != nil {
		return nil, mapImageError(err)
	}
	key, err := s.storeImage(img)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Add(ctx, userID, NewPhoto{StorageKey: key, Width: img.Width, Height: img.Height, SizeBytes: len(img.Full)}); err != nil {
		s.removeBlobs(key)
		if errors.Is(err, ErrRepositoryLimit) {
			return nil, ErrPhotoLimit
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) ([]users.PhotoView, error) {
	img, err := processImage(data)
	if err != nil {
		return nil, mapImageError(err)
	}
	key, err := s.storeImage(img)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, strings.ToLower(photoID), NewPhoto{StorageKey: key, Width: img.Width, Height: img.Height, SizeBytes: len(img.Full)})
	if err != nil {
		s.removeBlobs(key)
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	s.removeBlobs(oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]users.PhotoView, error) {
	key, err := s.repo.Delete(ctx, userID, strings.ToLower(photoID))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	s.removeBlobs(key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]users.PhotoView, error) {
	lowered := make([]string, len(ids))
	for i, id := range ids {
		lowered[i] = strings.ToLower(id)
	}
	if err := s.repo.Reorder(ctx, userID, lowered); err != nil {
		if errors.Is(err, ErrRepositoryBadReorder) {
			return nil, ErrBadReorder
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// SetMain moves a photo to position 0 and keeps the others in their order.
func (s *Service) SetMain(ctx context.Context, userID, photoID string) ([]users.PhotoView, error) {
	photoID = strings.ToLower(photoID)
	current, err := s.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(current))
	found := false
	for _, p := range current {
		if p.ID == photoID {
			found = true
			continue
		}
		ids = append(ids, p.ID)
	}
	if !found {
		return nil, ErrPhotoNotFound
	}
	return s.Reorder(ctx, userID, append([]string{photoID}, ids...))
}

// BeforeDelete implements auth.AccountHooks: files are removed after the account row is gone.
func (s *Service) BeforeDelete(ctx context.Context, userID string) (func(), error) {
	keys, err := s.repo.KeysForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() { s.removeBlobs(keys...) }, nil
}

// ---- serving ----

// Open returns the image bytes if the signed URL parameters are valid.
func (s *Service) Open(ctx context.Context, photoID, size, expRaw, sig string) (io.ReadSeekCloser, error) {
	if size != "full" && size != "thumb" {
		return nil, ErrBadSignature
	}
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || s.now().Unix() > exp {
		return nil, ErrBadSignature
	}
	if !hmac.Equal([]byte(sig), []byte(s.sign(photoID, size, exp))) {
		return nil, ErrBadSignature
	}
	photo, err := s.repo.Get(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	key := photo.StorageKey
	if size == "thumb" {
		key = thumbKey(key)
	}
	f, err := s.store.Open(key)
	if err != nil {
		if errors.Is(err, blob.ErrNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	return f, nil
}

// ---- helpers ----

func (s *Service) storeImage(img processed) (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	key := "p" + base64.RawURLEncoding.EncodeToString(raw)
	if err := s.store.Put(key, img.Full); err != nil {
		return "", err
	}
	if err := s.store.Put(thumbKey(key), img.Thumb); err != nil {
		_ = s.store.Delete(key)
		return "", err
	}
	return key, nil
}

func (s *Service) removeBlobs(keys ...string) {
	for _, key := range keys {
		if key == "" {
			continue
		}
		_ = s.store.Delete(key)
		_ = s.store.Delete(thumbKey(key))
	}
}

func thumbKey(key string) string { return key + "_t" }

func (s *Service) expiry() int64 {
	window := int64(urlWindow / time.Second)
	return (s.now().Unix()/window + 2) * window
}

func (s *Service) sign(photoID, size string, exp int64) string {
	mac := hmac.New(sha256.New, s.urlKey)
	fmt.Fprintf(mac, "%s|%s|%d", photoID, size, exp)
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

func (s *Service) url(photoID, size string, exp int64) string {
	return fmt.Sprintf("/photos/%s/file?size=%s&exp=%d&sig=%s", photoID, size, exp, s.sign(photoID, size, exp))
}

func mapImageError(err error) error {
	switch {
	case errors.Is(err, errImageTooLarge):
		return ErrImageTooLarge
	case errors.Is(err, errImageTooSmall):
		return ErrImageTooSmall
	default:
		return ErrUnsupportedType
	}
}
