package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"example.com/api/internal/platform/storage"
)

var (
	ErrNotFound  = errors.New("photo not found")
	ErrLimit     = errors.New("photo limit reached")
	ErrBadOrder  = errors.New("order must list each of your photos exactly once")
	ErrTemporary = errors.New("storage unavailable")
)

const MaxPhotos = 6

type Service struct {
	repo  Repository
	store storage.Store
	// decoding a 12 MP image takes ~100 MB: only a few run at once.
	slots chan struct{}
}

func NewService(repo Repository, store storage.Store) *Service {
	return &Service{repo: repo, store: store, slots: make(chan struct{}, 3)}
}

func (s *Service) process(ctx context.Context, r io.Reader) (Processed, error) {
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-ctx.Done():
		return Processed{}, ctx.Err()
	}
	return Process(r)
}

// Upload validates and stores a new photo at the end of the user's list.
func (s *Service) Upload(ctx context.Context, userID string, r io.Reader) (string, error) {
	processed, err := s.process(ctx, r)
	if err != nil {
		return "", err
	}
	key := newKey(userID)
	if err := s.store.Put(ctx, key, processed.Data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemporary, err)
	}
	photo, err := s.repo.Insert(ctx, NewPhoto{
		UserID: userID, StorageKey: key, SizeBytes: len(processed.Data), Width: processed.Width, Height: processed.Height,
	}, MaxPhotos)
	if err != nil {
		s.discard(key)
		if errors.Is(err, ErrRepositoryFull) {
			return "", ErrLimit
		}
		return "", err
	}
	return photo.ID, nil
}

// Replace swaps the image of an existing photo, keeping its position.
func (s *Service) Replace(ctx context.Context, userID, photoID string, r io.Reader) error {
	processed, err := s.process(ctx, r)
	if err != nil {
		return err
	}
	key := newKey(userID)
	if err := s.store.Put(ctx, key, processed.Data); err != nil {
		return fmt.Errorf("%w: %v", ErrTemporary, err)
	}
	oldKey, err := s.repo.Replace(ctx, photoID, userID, NewPhoto{
		UserID: userID, StorageKey: key, SizeBytes: len(processed.Data), Width: processed.Width, Height: processed.Height,
	})
	if err != nil {
		s.discard(key)
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.discard(oldKey)
	return nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, photoID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.discard(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	normalized := make([]string, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for i, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		if _, dup := seen[id]; dup {
			return ErrBadOrder
		}
		seen[id] = struct{}{}
		normalized[i] = id
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		if errors.Is(err, ErrRepositoryOrder) {
			return ErrBadOrder
		}
		return err
	}
	return nil
}

// MakeMain moves one photo to position 0 and keeps the others in their relative order.
func (s *Service) MakeMain(ctx context.Context, userID, photoID string) error {
	current, err := s.repo.ListIDs(ctx, userID)
	if err != nil {
		return err
	}
	photoID = strings.ToLower(photoID)
	order := []string{photoID}
	found := false
	for _, id := range current {
		if id == photoID {
			found = true
			continue
		}
		order = append(order, id)
	}
	if !found {
		return ErrNotFound
	}
	return s.Reorder(ctx, userID, order)
}

// Open returns the bytes of a photo; authorization is the caller's job (signed URL).
func (s *Service) Open(ctx context.Context, photoID string) (io.ReadSeekCloser, error) {
	photo, err := s.repo.Get(ctx, photoID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	f, err := s.store.Open(ctx, photo.StorageKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// StorageKeys lists every object owned by a user (used before account deletion).
func (s *Service) StorageKeys(ctx context.Context, userID string) ([]string, error) {
	return s.repo.StorageKeys(ctx, userID)
}

// PurgeObjects removes stored files best-effort once their rows are gone.
func (s *Service) PurgeObjects(keys []string) {
	for _, k := range keys {
		s.discard(k)
	}
}

func (s *Service) discard(key string) {
	if err := s.store.Delete(context.Background(), key); err != nil {
		log.Printf(`{"event":"photo_cleanup_failed","key":%q,"error":%q}`, key, err.Error())
	}
}

func newKey(userID string) string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err) // crypto/rand failing means the platform is unusable
	}
	return fmt.Sprintf("photos/%s/%s.jpg", userID, hex.EncodeToString(buf))
}
