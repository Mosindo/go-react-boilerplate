package users

import (
	"context"
	"errors"
	"log"

	"example.com/api/internal/platform/realtime"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// MediaRemover deletes stored blobs once their database rows are gone.
type MediaRemover interface {
	Delete(key string) error
}

type Service struct {
	repo      Repository
	media     MediaRemover
	publisher realtime.Publisher
}

func NewService(repo Repository, media MediaRemover, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, media: media, publisher: publisher}
}

// DeleteAccount permanently removes the member after re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	deleted, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	for _, key := range deleted.PhotoStorageKeys {
		if err := s.media.Delete(key); err != nil {
			log.Printf(`{"event":"account_delete_media_error","error":%q}`, err.Error())
		}
	}
	for _, partnerID := range deleted.FormerMatchIDs {
		s.publisher.Publish(partnerID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": userID}})
	}
	s.publisher.Disconnect(userID)
	return nil
}
