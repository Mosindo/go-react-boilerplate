package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"slices"

	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo    Repository
	store   storage.Storage
	signer  *media.Signer
	process func([]byte) (Processed, error)
}

func NewService(repo Repository, store storage.Storage, signer *media.Signer) *Service {
	return &Service{repo: repo, store: store, signer: signer, process: Process}
}

func (s *Service) toResponse(p Photo) PhotoResponse {
	return PhotoResponse{ID: p.ID, URL: s.signer.PhotoPath(p.ID), Position: p.Position, Width: p.Width, Height: p.Height}
}

func (s *Service) List(ctx context.Context, userID string) ([]PhotoResponse, error) {
	photos, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PhotoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, s.toResponse(p))
	}
	return out, nil
}

// newKey builds a random, non-guessable storage key unrelated to user input.
func newKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	h := hex.EncodeToString(b)
	return "photos/" + h[:2] + "/" + h + ".jpg", nil
}

func (s *Service) storeProcessed(ctx context.Context, raw []byte) (string, Processed, error) {
	processed, err := s.process(raw)
	if err != nil {
		return "", Processed{}, err
	}
	key, err := newKey()
	if err != nil {
		return "", Processed{}, err
	}
	if err := s.store.Put(ctx, key, processed.Data); err != nil {
		return "", Processed{}, err
	}
	return key, processed, nil
}

func (s *Service) discard(key string) {
	if err := s.store.Delete(context.Background(), key); err != nil {
		log.Printf(`{"event":"storage_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (PhotoResponse, error) {
	key, processed, err := s.storeProcessed(ctx, raw)
	if err != nil {
		return PhotoResponse{}, err
	}
	created, err := s.repo.Insert(ctx, Photo{UserID: userID, StorageKey: key, SizeBytes: len(processed.Data), Width: processed.Width, Height: processed.Height})
	if err != nil {
		s.discard(key)
		return PhotoResponse{}, err
	}
	return s.toResponse(created), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) error {
	key, processed, err := s.storeProcessed(ctx, raw)
	if err != nil {
		return err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, len(processed.Data), processed.Width, processed.Height)
	if err != nil {
		s.discard(key)
		return err
	}
	s.discard(oldKey)
	return nil
}

// Delete removes a photo. A complete profile must keep at least one photo.
func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID, true)
	if err != nil {
		return err
	}
	s.discard(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	seen := make([]string, 0, len(photoIDs))
	for _, id := range photoIDs {
		if slices.Contains(seen, id) {
			return ErrInvalidOrder
		}
		seen = append(seen, id)
	}
	return s.repo.Reorder(ctx, userID, photoIDs)
}

// Open serves a photo after its signed URL has been verified.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (io.ReadSeekCloser, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return nil, ErrNotFound
	}
	key, err := s.repo.StorageKey(ctx, photoID)
	if err != nil {
		return nil, err
	}
	f, err := s.store.Open(ctx, key)
	if err != nil {
		return nil, ErrNotFound
	}
	return f, nil
}

// PrepareAccountDeletion collects the user's files before the rows vanish
// and returns a cleanup to run once the account is deleted.
func (s *Service) PrepareAccountDeletion(ctx context.Context, userID string) (func(), error) {
	keys, err := s.repo.StorageKeysForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() {
		for _, key := range keys {
			s.discard(key)
		}
	}, nil
}
