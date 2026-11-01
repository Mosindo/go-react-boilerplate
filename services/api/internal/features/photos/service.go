package photos

import (
	"context"
	"errors"
	"os"
)

type Service struct {
	repo    Repository
	storage *DiskStorage
}

func NewService(repo Repository, storage *DiskStorage) *Service {
	return &Service{repo: repo, storage: storage}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

// Add valide, recompresse et stocke une photo. replaceID optionnel remplace une photo existante.
func (s *Service) Add(ctx context.Context, userID string, data []byte, replaceID string) (Photo, error) {
	processed, w, h, err := Process(data)
	if err != nil {
		return Photo{}, err
	}
	key, err := newKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.storage.Save(key, processed); err != nil {
		return Photo{}, err
	}
	p, oldKey, err := s.repo.Insert(ctx, userID, File{StorageKey: key, ContentType: "image/jpeg"}, w, h, len(processed), replaceID)
	if err != nil {
		s.storage.Remove([]string{key}) // pas de fichier orphelin
		return Photo{}, err
	}
	if oldKey != "" {
		s.storage.Remove([]string{oldKey})
	}
	return p, nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return err
	}
	s.storage.Remove([]string{key})
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}

// MakePrimary place la photo en premiere position en conservant l'ordre des autres.
func (s *Service) MakePrimary(ctx context.Context, userID, photoID string) error {
	list, err := s.repo.List(ctx, userID)
	if err != nil {
		return err
	}
	ids := []string{photoID}
	found := false
	for _, p := range list {
		if p.ID == photoID {
			found = true
			continue
		}
		ids = append(ids, p.ID)
	}
	if !found {
		return ErrNotFound
	}
	return s.repo.Reorder(ctx, userID, ids)
}

// Open ouvre le fichier d'une photo si viewerID a le droit de la voir.
func (s *Service) Open(ctx context.Context, viewerID, photoID string) (*os.File, error) {
	f, err := s.repo.FileForViewer(ctx, viewerID, photoID)
	if err != nil {
		return nil, err
	}
	file, err := s.storage.Open(f.StorageKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return file, nil
}

// RemoveFiles implemente auth.FileRemover (suppression de compte).
func (s *Service) RemoveFiles(keys []string) { s.storage.Remove(keys) }
