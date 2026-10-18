package photos

import (
	"context"
	"errors"
	"io"
	"time"

	"example.com/api/internal/platform/storage"
	"github.com/google/uuid"
)

type Service struct {
	repo    Repository
	storage storage.Storage
	signer  *Signer
	now     func() time.Time
}

func NewService(repo Repository, files storage.Storage, signer *Signer) *Service {
	return &Service{repo: repo, storage: files, signer: signer, now: time.Now}
}

func (s *Service) toPhoto(p storedPhoto) Photo {
	now := s.now()
	return Photo{
		ID:       p.ID,
		Position: p.Position,
		URL:      s.signer.URL(p.ID, variantFull, now),
		ThumbURL: s.signer.URL(p.ID, variantThumb, now),
		Width:    p.Width,
		Height:   p.Height,
	}
}

// ListForUsers returns photos grouped by user id, loaded with a single query.
// Callers must have already checked that the viewer may see these users.
func (s *Service) ListForUsers(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	stored, err := s.repo.ListByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]Photo, len(userIDs))
	for _, p := range stored {
		out[p.UserID] = append(out[p.UserID], s.toPhoto(p))
	}
	return out, nil
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	m, err := s.ListForUsers(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	if m[userID] == nil {
		return []Photo{}, nil
	}
	return m[userID], nil
}

func (s *Service) store(userID string, data []byte) (storedPhoto, error) {
	img, err := processImage(data)
	if err != nil {
		return storedPhoto{}, err
	}
	base := userID + "/" + uuid.NewString()
	p := storedPhoto{
		UserID:     userID,
		StorageKey: base + ".jpg",
		ThumbKey:   base + "_thumb.jpg",
		Width:      img.Width,
		Height:     img.Height,
		SizeBytes:  len(img.Full),
	}
	if err := s.storage.Put(p.StorageKey, img.Full); err != nil {
		return storedPhoto{}, err
	}
	if err := s.storage.Put(p.ThumbKey, img.Thumb); err != nil {
		_ = s.storage.Delete(p.StorageKey)
		return storedPhoto{}, err
	}
	return p, nil
}

func (s *Service) Add(ctx context.Context, userID string, data []byte) (Photo, error) {
	p, err := s.store(userID, data)
	if err != nil {
		return Photo{}, err
	}
	saved, err := s.repo.Insert(ctx, p)
	if err != nil {
		_ = s.storage.Delete(p.StorageKey, p.ThumbKey)
		return Photo{}, err
	}
	return s.toPhoto(saved), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) (Photo, error) {
	p, err := s.store(userID, data)
	if err != nil {
		return Photo{}, err
	}
	old, updated, err := s.repo.Replace(ctx, userID, photoID, p)
	if err != nil {
		_ = s.storage.Delete(p.StorageKey, p.ThumbKey)
		return Photo{}, err
	}
	_ = s.storage.Delete(old.StorageKey, old.ThumbKey)
	return s.toPhoto(updated), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	old, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	_ = s.storage.Delete(old.StorageKey, old.ThumbKey)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}

// Open serves a file for a signed URL.
func (s *Service) Open(ctx context.Context, photoID, variant, exp, sig string) (io.ReadSeekCloser, error) {
	if variant != variantFull && variant != variantThumb {
		return nil, ErrNotFound
	}
	if !s.signer.Verify(photoID, variant, exp, sig, s.now()) {
		return nil, ErrNotFound
	}
	full, thumb, err := s.repo.GetKeys(ctx, photoID)
	if err != nil {
		return nil, err
	}
	key := full
	if variant == variantThumb {
		key = thumb
	}
	f, err := s.storage.Open(key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	}
	return f, err
}
