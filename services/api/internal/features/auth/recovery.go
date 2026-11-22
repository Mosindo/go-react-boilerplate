package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"
	"unicode"

	"example.com/api/internal/platform/mailer"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrWeakPassword     = errors.New("password must be 8-128 characters and contain a letter and a digit")
	ErrInvalidResetCode = errors.New("invalid or expired code")
	ErrWrongPassword    = errors.New("current password is incorrect")
)

const (
	resetCodeTTL         = 15 * time.Minute
	maxResetCodeAttempts = 5
)

// ValidatePassword enforces a minimal, user-friendly password policy.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
		return ErrWeakPassword
	}
	var letter, digit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !letter || !digit {
		return ErrWeakPassword
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func generateResetCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// hashResetCode binds the code to the user so a leaked hash table cannot be
// replayed across accounts.
func hashResetCode(userID, code string) string {
	sum := sha256.Sum256([]byte(userID + ":" + code))
	return hex.EncodeToString(sum[:])
}

// ForgotPassword sends a one-time 6-digit code by e-mail. It never reveals
// whether the address is registered.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, ErrRepositoryNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreateResetToken(ctx, stored.User.ID, hashResetCode(stored.User.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	if s.mailer == nil {
		return errors.New("no mailer configured")
	}
	err = s.mailer.Send(ctx, mailer.Message{
		To:      stored.User.Email,
		Subject: "Votre code de réinitialisation Lueur",
		Body: "Bonjour,\n\nVotre code de réinitialisation est : " + code +
			"\nIl expire dans 15 minutes.\n\nSi vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n",
	})
	if err != nil {
		log.Printf(`{"event":"reset_mail_failed","error":%q}`, err.Error())
	}
	return nil
}

// ResetPassword checks the code (max 5 attempts), sets the new password and
// signs the account out everywhere.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrInvalidResetCode
	}
	if err != nil {
		return err
	}
	token, err := s.repo.ActiveResetToken(ctx, stored.User.ID)
	if errors.Is(err, ErrRepositoryResetTokenGone) {
		return ErrInvalidResetCode
	}
	if err != nil {
		return err
	}
	if token.Attempts >= maxResetCodeAttempts {
		return ErrInvalidResetCode
	}
	expected := hashResetCode(stored.User.ID, code)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(token.CodeHash)) != 1 {
		if err := s.repo.IncrementResetAttempts(ctx, token.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumeResetToken(ctx, token.ID, stored.User.ID, hash); err != nil {
		if errors.Is(err, ErrRepositoryResetTokenGone) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

func (s *Service) verifyPassword(ctx context.Context, userID, password string) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return ErrWrongPassword
	}
	return nil
}

// ChangePassword keeps the current session and revokes all the others.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := ValidatePassword(next); err != nil {
		return err
	}
	if err := s.verifyPassword(ctx, userID, current); err != nil {
		return err
	}
	hash, err := hashPassword(next)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	return s.repo.RevokeSessions(ctx, userID, sessionID)
}

// DeleteAccount permanently erases the account after password confirmation.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return err
	}
	deleted, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if s.onDeleted != nil {
		s.onDeleted(deleted)
	}
	return nil
}

// SessionActive backs middleware.SessionValidator.
func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.SessionActive(ctx, sessionID, userID)
}
