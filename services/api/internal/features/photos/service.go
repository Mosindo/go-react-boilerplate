package photos

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"slices"

	"example.com/api/internal/platform/mediaurl"
)

const MaxPhotos = 6

var (
	ErrLimit        = errors.New("you can have at most 6 photos")
	ErrNotFound     = errors.New("photo not found")
	ErrInvalidOrder = errors.New("order must list every photo exactly once")
	ErrBadSignature = errors.New("invalid or expired media link")
)

type Service struct {
	repo   Repository
	store  Storage
	signer *mediaurl.Signer
}

func NewService(repo Repository, store Storage, signer *mediaurl.Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer}
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func fullKey(id string) string  { return id + ".jpg" }
func thumbKey(id string) string { return id + "_thumb.jpg" }

func (s *Service) put(id string, p processed) error {
	if err := s.store.Put(fullKey(id), p.full); err != nil {
		return err
	}
	if err := s.store.Put(thumbKey(id), p.thumb); err != nil {
		_ = s.store.Delete(fullKey(id))
		return err
	}
	return nil
}

func (s *Service) remove(id string) {
	_ = s.store.Delete(fullKey(id))
	_ = s.store.Delete(thumbKey(id))
}

// Upload validates, sanitises and stores a new photo appended at the end.
func (s *Service) Upload(ctx context.Context, userID string, data []byte) (string, error) {
	p, err := process(data)
	if err != nil {
		return "", err
	}
	id, err := newID()
	if err != nil {
		return "", err
	}
	if err := s.put(id, p); err != nil {
		return "", err
	}
	if err := s.repo.Insert(ctx, id, userID, p.width, p.height, len(p.full), MaxPhotos); err != nil {
		s.remove(id)
		if errors.Is(err, ErrRepositoryLimit) {
			return "", ErrLimit
		}
		return "", err
	}
	return id, nil
}

// Replace swaps the image of an owned photo, keeping its position.
func (s *Service) Replace(ctx context.Context, userID, photoID string, data []byte) error {
	ids, err := s.repo.List(ctx, userID)
	if err != nil {
		return err
	}
	if !slices.Contains(ids, photoID) {
		return ErrNotFound
	}
	p, err := process(data)
	if err != nil {
		return err
	}
	if err := s.put(photoID, p); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, photoID, userID, p.width, p.height, len(p.full)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	if err := s.repo.Delete(ctx, photoID, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.remove(photoID)
	return nil
}

// Reorder sets the display order; the first id becomes the primary photo.
func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 || len(ids) > MaxPhotos {
		return ErrInvalidOrder
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return ErrInvalidOrder
		}
		seen[id] = struct{}{}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrRepositoryOrder) {
			return ErrInvalidOrder
		}
		return err
	}
	return nil
}

// PrepareAccountDeletion returns a function that erases the user's files; call it
// once the database rows are gone.
func (s *Service) PrepareAccountDeletion(ctx context.Context, userID string) func() {
	ids, err := s.repo.List(ctx, userID)
	if err != nil {
		return func() {}
	}
	return func() {
		for _, id := range ids {
			s.remove(id)
		}
	}
}

// Open verifies a signed media link and returns the JPEG bytes.
func (s *Service) Open(photoID, size, exp, sig string) ([]byte, error) {
	if size != "full" && size != "thumb" {
		return nil, ErrNotFound
	}
	if !s.signer.Verify(photoID, size, exp, sig) {
		return nil, ErrBadSignature
	}
	key := fullKey(photoID)
	if size == "thumb" {
		key = thumbKey(photoID)
	}
	b, err := s.store.Get(key)
	if errors.Is(err, ErrStorageNotFound) {
		return nil, ErrNotFound
	}
	return b, err
}
