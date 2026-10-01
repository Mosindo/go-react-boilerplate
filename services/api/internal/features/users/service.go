package users

import (
	"context"
	"errors"
	"log"

	"example.com/api/internal/platform/storage"
	"golang.org/x/crypto/bcrypt"
)

var ErrWrongPassword = errors.New("wrong password")

type Disconnector interface {
	Disconnect(userID string)
}

type Service struct {
	repo  Repository
	store storage.Store
	hub   Disconnector
}

func NewService(repo Repository, store storage.Store, hub Disconnector) *Service {
	return &Service{repo: repo, store: store, hub: hub}
}

// DeleteAccount erases the account and every file it owns.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	hash, err := s.repo.PasswordHash(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrWrongPassword
	}
	keys, err := s.repo.PhotoKeys(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, userID); err != nil {
		return err
	}
	for _, key := range keys {
		if err := s.store.Delete(key); err != nil {
			log.Printf(`{"event":"account_delete_file_cleanup_failed","error":%q}`, err.Error())
		}
	}
	s.hub.Disconnect(userID)
	return nil
}
