package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"

	"example.com/api/internal/platform/storage"
)

type URLSigner interface {
	Sign(photoID string) string
	Verify(photoID, exp, sig string) bool
}

type Service struct {
	repo    Repository
	storage storage.Storage
	signer  URLSigner
}

func NewService(repo Repository, store storage.Storage, signer URLSigner) *Service {
	return &Service{repo: repo, storage: store, signer: signer}
}

func (s *Service) toResponse(photos []StoredPhoto) PhotosResponse {
	out := make([]PhotoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, PhotoResponse{ID: p.ID, Position: p.Position, URL: s.signer.Sign(p.ID)})
	}
	return PhotosResponse{Photos: out}
}

func (s *Service) List(ctx context.Context, userID string) (PhotosResponse, error) {
	photos, err := s.repo.List(ctx, userID)
	if err != nil {
		return PhotosResponse{}, err
	}
	return s.toResponse(photos), nil
}

func newStorageKey(userID string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "photos/" + strings.ToLower(userID) + "/" + hex.EncodeToString(buf) + ".jpg", nil
}

// Upload validates, re-encodes and stores a new photo at the next position.
func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (PhotosResponse, error) {
	img, err := ProcessImage(raw)
	if err != nil {
		return PhotosResponse{}, err
	}
	key, err := newStorageKey(userID)
	if err != nil {
		return PhotosResponse{}, err
	}
	if err := s.storage.Put(ctx, key, img.Data); err != nil {
		return PhotosResponse{}, err
	}
	if _, err := s.repo.Create(ctx, userID, key, img); err != nil {
		s.purge(ctx, key)
		return PhotosResponse{}, err
	}
	return s.List(ctx, userID)
}

// Replace swaps the image of an existing photo while keeping its position.
func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (PhotosResponse, error) {
	img, err := ProcessImage(raw)
	if err != nil {
		return PhotosResponse{}, err
	}
	key, err := newStorageKey(userID)
	if err != nil {
		return PhotosResponse{}, err
	}
	if err := s.storage.Put(ctx, key, img.Data); err != nil {
		return PhotosResponse{}, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, img)
	if err != nil {
		s.purge(ctx, key)
		return PhotosResponse{}, err
	}
	s.purge(ctx, oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) (PhotosResponse, error) {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return PhotosResponse{}, err
	}
	s.purge(ctx, key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) (PhotosResponse, error) {
	normalized := make([]string, 0, len(photoIDs))
	for _, id := range photoIDs {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(id)))
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		return PhotosResponse{}, err
	}
	return s.List(ctx, userID)
}

// Open returns the stored object for a signed media URL.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (storage.Object, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return nil, ErrPhotoNotFound
	}
	key, err := s.repo.GetStorageKey(ctx, photoID)
	if err != nil {
		return nil, err
	}
	obj, err := s.storage.Open(ctx, key)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrPhotoNotFound
		}
		return nil, err
	}
	return obj, nil
}

// StorageKeysForUser and PurgeObjects implement auth.StorageCleaner.
func (s *Service) StorageKeysForUser(ctx context.Context, userID string) ([]string, error) {
	return s.repo.ListStorageKeys(ctx, userID)
}

func (s *Service) PurgeObjects(ctx context.Context, keys []string) {
	for _, key := range keys {
		s.purge(ctx, key)
	}
}

func (s *Service) purge(ctx context.Context, key string) {
	if err := s.storage.Delete(context.WithoutCancel(ctx), key); err != nil {
		log.Printf(`{"event":"photo_purge_error","key":%q,"error":%q}`, key, err.Error())
	}
}
