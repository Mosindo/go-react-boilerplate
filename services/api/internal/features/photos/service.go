package photos

import (
	"context"
	"log"

	"example.com/api/internal/platform/media"
)

type Service struct {
	repo    Repository
	storage media.Storage
	signer  *media.Signer
}

func NewService(repo Repository, storage media.Storage, signer *media.Signer) *Service {
	return &Service{repo: repo, storage: storage, signer: signer}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.views(stored), nil
}

// Upload validates and re-encodes the image before storing it.
func (s *Service) Upload(ctx context.Context, userID string, raw []byte) (Photo, error) {
	processed, err := media.Process(raw)
	if err != nil {
		return Photo{}, err
	}
	key, err := s.storage.Save(processed.Data)
	if err != nil {
		return Photo{}, err
	}
	stored, err := s.repo.Insert(ctx, userID, key, processed.Width, processed.Height, len(processed.Data))
	if err != nil {
		s.removeFile(key)
		return Photo{}, err
	}
	return s.view(stored), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) ([]Photo, error) {
	processed, err := media.Process(raw)
	if err != nil {
		return nil, err
	}
	key, err := s.storage.Save(processed.Data)
	if err != nil {
		return nil, err
	}
	oldKey, err := s.repo.Replace(ctx, userID, photoID, key, processed.Width, processed.Height, len(processed.Data))
	if err != nil {
		s.removeFile(key)
		return nil, err
	}
	s.removeFile(oldKey)
	return s.List(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) ([]Photo, error) {
	current, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(current) == 1 && current[0].ID == photoID {
		discoverable, err := s.repo.IsDiscoverable(ctx, userID)
		if err != nil {
			return nil, err
		}
		if discoverable {
			return nil, ErrLastPhoto
		}
	}
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return nil, err
	}
	s.removeFile(key)
	return s.List(ctx, userID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.List(ctx, userID)
}

// ResolveMedia verifies a signed URL and returns the file path to serve.
func (s *Service) ResolveMedia(ctx context.Context, key, exp, sig string) (string, error) {
	if !media.ValidKey(key) || !s.signer.Verify(key, exp, sig) {
		return "", ErrMediaNotFound
	}
	exists, err := s.repo.ExistsByKey(ctx, key)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrMediaNotFound
	}
	return s.storage.Path(key)
}

// RemoveFiles deletes stored files (used after account deletion).
func (s *Service) RemoveFiles(keys []string) {
	for _, k := range keys {
		s.removeFile(k)
	}
}

func (s *Service) removeFile(key string) {
	if err := s.storage.Delete(key); err != nil {
		log.Printf(`{"event":"media_delete_failed","error":%q}`, err.Error())
	}
}

func (s *Service) view(p StoredPhoto) Photo {
	return Photo{ID: p.ID, URL: s.signer.URL(p.StorageKey), Position: p.Position, Width: p.Width, Height: p.Height}
}

func (s *Service) views(stored []StoredPhoto) []Photo {
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, s.view(p))
	}
	return out
}
