package photos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
)

type Service struct {
	repo    Repository
	storage storage.Storage
	signer  *storage.Signer
}

func NewService(repo Repository, store storage.Storage, signer *storage.Signer) *Service {
	return &Service{repo: repo, storage: store, signer: signer}
}

func (s *Service) view(p storedPhoto) Photo {
	return Photo{ID: p.ID, URL: s.signer.URL(p.ID, "full"), ThumbURL: s.signer.URL(p.ID, "thumb"), Position: p.Position}
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	stored, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, s.view(p))
	}
	return out, nil
}

// Upload validates, re-encodes and stores a new photo.
func (s *Service) Upload(ctx context.Context, userID string, file io.Reader) (Photo, error) {
	img, err := readAndProcess(file)
	if err != nil {
		return Photo{}, err
	}
	id, err := newID()
	if err != nil {
		return Photo{}, err
	}
	key, thumbKey := userID+"/"+id+".jpg", userID+"/"+id+"_t.jpg"
	if err := s.putBoth(key, thumbKey, img); err != nil {
		return Photo{}, err
	}
	stored, err := s.repo.Insert(ctx, userID, id, storedPhoto{Key: key, ThumbKey: thumbKey}, img.Width, img.Height, len(img.Full))
	if err != nil {
		s.deleteQuiet(key, thumbKey)
		if errors.Is(err, ErrLimit) {
			return Photo{}, perr.Conflict("photo_limit", "you can upload up to 6 photos")
		}
		return Photo{}, err
	}
	return s.view(stored), nil
}

// Replace swaps the content of an existing photo, keeping its position.
func (s *Service) Replace(ctx context.Context, userID, photoID string, file io.Reader) (Photo, error) {
	img, err := readAndProcess(file)
	if err != nil {
		return Photo{}, err
	}
	id, err := newID()
	if err != nil {
		return Photo{}, err
	}
	key, thumbKey := userID+"/"+id+".jpg", userID+"/"+id+"_t.jpg"
	if err := s.putBoth(key, thumbKey, img); err != nil {
		return Photo{}, err
	}
	old, updated, err := s.repo.Replace(ctx, userID, photoID, key, thumbKey, img.Width, img.Height, len(img.Full))
	if err != nil {
		s.deleteQuiet(key, thumbKey)
		return Photo{}, mapNotFound(err)
	}
	s.deleteQuiet(old.Key, old.ThumbKey)
	return s.view(updated), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	if !validUUID(photoID) {
		return perr.NotFound("photo_not_found", "photo not found")
	}
	old, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		return mapNotFound(err)
	}
	s.deleteQuiet(old.Key, old.ThumbKey)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	for _, id := range ids {
		if !validUUID(id) {
			return nil, perr.Invalid("invalid_order", "photoIds must list exactly your photos")
		}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, perr.Invalid("invalid_order", "photoIds must list exactly your photos")
		}
		return nil, err
	}
	return s.List(ctx, userID)
}

// Open resolves a signed media URL to a stream. The signature is the authorisation.
func (s *Service) Open(ctx context.Context, photoID, size, exp, sig string) (io.ReadCloser, error) {
	notFound := perr.NotFound("media_not_found", "not found")
	if (size != "full" && size != "thumb") || !validUUID(photoID) || !s.signer.Verify(photoID, size, exp, sig) {
		return nil, notFound
	}
	key, thumbKey, err := s.repo.Keys(ctx, photoID)
	if err != nil {
		return nil, notFound
	}
	if size == "thumb" {
		key = thumbKey
	}
	rc, err := s.storage.Open(key)
	if err != nil {
		return nil, notFound
	}
	return rc, nil
}

func readAndProcess(file io.Reader) (processed, error) {
	raw, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		return processed{}, perr.Invalid("invalid_upload", "could not read the uploaded file")
	}
	return processImage(raw)
}

func (s *Service) putBoth(key, thumbKey string, img processed) error {
	if err := s.storage.Put(key, img.Full); err != nil {
		return err
	}
	if err := s.storage.Put(thumbKey, img.Thumb); err != nil {
		s.deleteQuiet(key)
		return err
	}
	return nil
}

func (s *Service) deleteQuiet(keys ...string) {
	for _, k := range keys {
		if err := s.storage.Delete(k); err != nil {
			log.Printf(`{"event":"storage_error","operation":"delete","error":%q}`, err.Error())
		}
	}
}

func mapNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return perr.NotFound("photo_not_found", "photo not found")
	}
	return err
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:], nil
}

func validUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, r := range v {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'):
			return false
		}
	}
	return true
}
