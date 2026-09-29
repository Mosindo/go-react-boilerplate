package photos

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"

	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/urlsign"
)

var ErrInvalidIDs = errors.New("photoIds must be valid photo ids")

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsUUID reports whether s is a lower-case canonical UUID.
func IsUUID(s string) bool { return uuidRe.MatchString(s) }

type Service struct {
	repo    Repository
	storage storage.Storage
	signer  *urlsign.Signer
	// slots bounds concurrent image decodes: a decoded 40 MP image is hundreds of MB.
	slots chan struct{}
}

func NewService(repo Repository, store storage.Storage, signer *urlsign.Signer) *Service {
	return &Service{repo: repo, storage: store, signer: signer, slots: make(chan struct{}, 4)}
}

func (s *Service) toPhoto(p storedPhoto) Photo {
	return Photo{ID: p.ID, URL: s.signer.PhotoURL(p.ID), Position: p.Position}
}

func newKey(userID string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("photos/%s/%x-%x-%x-%x-%x.jpg", userID, b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// Upload validates, normalises and stores a new photo for the user.
func (s *Service) Upload(ctx context.Context, userID string, data []byte) (Photo, error) {
	if n, err := s.repo.Count(ctx, userID); err != nil {
		return Photo{}, err
	} else if n >= MaxPhotosPerUser {
		return Photo{}, ErrLimitReached
	}

	select {
	case s.slots <- struct{}{}:
	case <-ctx.Done():
		return Photo{}, ctx.Err()
	}
	out, w, h, err := ProcessImage(data)
	<-s.slots
	if err != nil {
		return Photo{}, err
	}

	key, err := newKey(userID)
	if err != nil {
		return Photo{}, err
	}
	if _, err := s.storage.Put(ctx, key, bytes.NewReader(out)); err != nil {
		return Photo{}, err
	}
	p, err := s.repo.Add(ctx, userID, key, w, h, len(out))
	if err != nil {
		// The row was not created: do not leave an orphan object behind.
		if derr := s.storage.Delete(context.WithoutCancel(ctx), key); derr != nil {
			log.Printf(`{"event":"photo_orphan_cleanup_failed"}`)
		}
		return Photo{}, err
	}
	return s.toPhoto(p), nil
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(rows))
	for _, p := range rows {
		out = append(out, s.toPhoto(p))
	}
	return out, nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	if len(ids) == 0 || len(ids) > MaxPhotosPerUser {
		return nil, ErrNotAPermutation
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if !IsUUID(id) {
			return nil, ErrInvalidIDs
		}
		if _, dup := seen[id]; dup {
			return nil, ErrNotAPermutation
		}
		seen[id] = struct{}{}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	if !IsUUID(photoID) {
		return ErrNotFound
	}
	key, err := s.repo.Remove(ctx, userID, photoID)
	if err != nil {
		return err
	}
	if err := s.storage.Delete(context.WithoutCancel(ctx), key); err != nil {
		log.Printf(`{"event":"photo_delete_file_failed"}`)
	}
	return nil
}

// Open verifies the signature and returns the stored bytes. Every failure is ErrNotFound so
// callers cannot distinguish a bad signature from a missing photo.
func (s *Service) Open(ctx context.Context, photoID string, exp int64, sig string) (io.ReadCloser, int64, error) {
	if !IsUUID(photoID) || !s.signer.Verify(photoID, exp, sig) {
		return nil, 0, ErrNotFound
	}
	key, size, err := s.repo.File(ctx, photoID)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	rc, err := s.storage.Open(ctx, key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, err
	}
	return rc, size, nil
}
