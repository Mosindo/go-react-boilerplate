package account

import (
	"context"
	"errors"
	"log"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/storage"
)

var ErrWrongPassword = errors.New("incorrect password")

type Service struct {
	repo    Repository
	storage storage.Storage
	now     func() time.Time
}

func NewService(repo Repository, store storage.Storage) *Service {
	return &Service{repo: repo, storage: store, now: time.Now}
}

func (s *Service) Me(ctx context.Context, userID string) (auth.MeResponse, error) {
	return s.repo.Me(ctx, userID, s.now())
}

func (s *Service) verify(ctx context.Context, userID, password string) error {
	hash, err := s.repo.PasswordHash(ctx, userID)
	if err != nil {
		return err
	}
	if !auth.CheckPassword(hash, password) {
		return ErrWrongPassword
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := auth.ValidatePassword(next); err != nil {
		return err
	}
	if err := s.verify(ctx, userID, current); err != nil {
		return err
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return err
	}
	return s.repo.ChangePassword(ctx, userID, hash, sessionID)
}

// Delete permanently removes the account. Files are removed after the transaction committed,
// best effort: an orphaned object is preferable to a row pointing at a missing file.
func (s *Service) Delete(ctx context.Context, userID, password string) error {
	if err := s.verify(ctx, userID, password); err != nil {
		return err
	}
	keys, err := s.repo.Delete(ctx, userID)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := s.storage.Delete(context.WithoutCancel(ctx), k); err != nil {
			log.Printf(`{"event":"account_delete_file_failed","user_id":%q}`, userID)
		}
	}
	return nil
}
