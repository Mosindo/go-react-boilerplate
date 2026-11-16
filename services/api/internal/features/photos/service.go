package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"strings"

	"example.com/api/internal/platform/imaging"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo    Repository
	storage storage.Storage
	signer  *media.Signer
}

func NewService(repo Repository, store storage.Storage, signer *media.Signer) *Service {
	return &Service{repo: repo, storage: store, signer: signer}
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

func (s *Service) processAndStore(ctx context.Context, raw []byte) (string, imaging.Result, error) {
	img, err := imaging.Process(raw)
	if err != nil {
		return "", imaging.Result{}, err
	}
	key, err := newStorageKey()
	if err != nil {
		return "", imaging.Result{}, err
	}
	if err := s.storage.Put(ctx, key, img.Data); err != nil {
		return "", imaging.Result{}, err
	}
	return key, img, nil
}

func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (PhotoResponse, error) {
	key, img, err := s.processAndStore(ctx, raw)
	if err != nil {
		return PhotoResponse{}, err
	}
	photo, err := s.repo.Insert(ctx, userID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.deleteFile(ctx, key)
		return PhotoResponse{}, err
	}
	return s.toResponse(photo), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (PhotoResponse, error) {
	key, img, err := s.processAndStore(ctx, raw)
	if err != nil {
		return PhotoResponse{}, err
	}
	oldKey, photo, err := s.repo.Replace(ctx, userID, photoID, key, img.Width, img.Height, len(img.Data))
	if err != nil {
		s.deleteFile(ctx, key)
		return PhotoResponse{}, err
	}
	s.deleteFile(ctx, oldKey)
	return s.toResponse(photo), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.deleteFile(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, photoIDs []string) ([]PhotoResponse, error) {
	normalized := make([]string, len(photoIDs))
	for i, id := range photoIDs {
		normalized[i] = strings.ToLower(id)
	}
	if err := s.repo.Reorder(ctx, userID, normalized); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open serves a photo after verifying its signed URL.
func (s *Service) Open(ctx context.Context, photoID, exp, sig string) (io.ReadSeekCloser, error) {
	if !s.signer.Verify(photoID, exp, sig) {
		return nil, ErrNotFound
	}
	key, err := s.repo.StorageKey(ctx, photoID)
	if err != nil {
		return nil, err
	}
	f, err := s.storage.Open(ctx, key)
	if err != nil {
		return nil, ErrNotFound
	}
	return f, nil
}

func (s *Service) deleteFile(ctx context.Context, key string) {
	if err := s.storage.Delete(ctx, key); err != nil {
		log.Printf(`{"event":"photo_file_delete_failed","key":%q,"error":%q}`, key, err.Error())
	}
}

func (s *Service) toResponse(p Photo) PhotoResponse {
	return PhotoResponse{ID: p.ID, URL: s.signer.URL(p.ID), Position: p.Position, Width: p.Width, Height: p.Height}
}

func newStorageKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	h := hex.EncodeToString(b)
	return "photos/" + h[:2] + "/" + h + ".jpg", nil
}
