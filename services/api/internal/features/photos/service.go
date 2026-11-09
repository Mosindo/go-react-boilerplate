package photos

import (
	"context"
	"io"
	"log"
)

type Service struct {
	repo    Repository
	storage Storage
}

func NewService(repo Repository, storage Storage) *Service {
	return &Service{repo: repo, storage: storage}
}

func toPhoto(p StoredPhoto) Photo {
	return Photo{ID: p.ID, URL: "/photos/" + p.ID, Position: p.Position, Width: p.Width, Height: p.Height}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, toPhoto(p))
	}
	return out, nil
}

// store validates, sanitizes and writes the image, returning its key and metadata.
func (s *Service) store(raw io.Reader) (key string, w, h, size int, err error) {
	buf, err := io.ReadAll(io.LimitReader(raw, MaxUploadSize+1))
	if err != nil {
		return "", 0, 0, 0, err
	}
	data, w, h, err := ProcessImage(buf)
	if err != nil {
		return "", 0, 0, 0, err
	}
	key, err = NewKey()
	if err != nil {
		return "", 0, 0, 0, err
	}
	if err := s.storage.Save(key, data); err != nil {
		return "", 0, 0, 0, err
	}
	return key, w, h, len(data), nil
}

func (s *Service) Upload(ctx context.Context, userID string, raw io.Reader) (Photo, error) {
	key, w, h, size, err := s.store(raw)
	if err != nil {
		return Photo{}, err
	}
	p, err := s.repo.Add(ctx, userID, key, w, h, size)
	if err != nil {
		s.cleanup(key)
		return Photo{}, err
	}
	return toPhoto(p), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw io.Reader) error {
	key, w, h, size, err := s.store(raw)
	if err != nil {
		return err
	}
	old, err := s.repo.Replace(ctx, userID, photoID, key, w, h, size)
	if err != nil {
		s.cleanup(key)
		return err
	}
	s.cleanup(old)
	return nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Remove(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.cleanup(key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}

// MakePrimary moves a photo to position 0, keeping the relative order of the others.
func (s *Service) MakePrimary(ctx context.Context, userID, photoID string) error {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return err
	}
	ids := []string{}
	found := false
	for _, p := range stored {
		if p.ID == photoID {
			found = true
			continue
		}
		ids = append(ids, p.ID)
	}
	if !found {
		return ErrNotFound
	}
	return s.repo.Reorder(ctx, userID, append([]string{photoID}, ids...))
}

func (s *Service) Read(ctx context.Context, viewerID, photoID string) ([]byte, error) {
	key, err := s.repo.KeyForViewer(ctx, viewerID, photoID)
	if err != nil {
		return nil, err
	}
	data, err := s.storage.Read(key)
	if err != nil {
		return nil, ErrNotFound
	}
	return data, nil
}

func (s *Service) cleanup(key string) {
	if err := s.storage.Delete(key); err != nil {
		log.Printf("photos: could not delete %s: %v", key, err)
	}
}
