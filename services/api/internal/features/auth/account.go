package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"example.com/api/internal/platform/mailer"

	platformerrors "example.com/api/internal/platform/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	// bcrypt only hashes the first 72 bytes, so longer passwords are rejected
	// instead of being silently truncated.
	maxPasswordBytes = 72
	minPasswordRunes = 8
	resetCodeTTL     = 15 * time.Minute
)

var (
	ErrWeakPassword      = platformerrors.BadRequest("invalid_password", "password must be between 8 and 72 bytes")
	ErrWrongPassword     = platformerrors.Forbidden("wrong_password", "password is incorrect")
	ErrInvalidEmail      = platformerrors.BadRequest("invalid_email", "email address is not valid")
	ErrInvalidResetCode  = platformerrors.BadRequest("invalid_reset_code", "the code is invalid or has expired")
	errNoMailerAvailable = errors.New("mailer not configured")
)

// AccountHooks lets other modules release what they own when an account is deleted.
// BeforeDelete runs before the database row is removed and returns a cleanup
// function that runs once the deletion succeeded (e.g. removing photo files).
type AccountHooks interface {
	BeforeDelete(ctx context.Context, userID string) (cleanup func(), err error)
}

// SetMailer configures the account-recovery email sender.
func (s *Service) SetMailer(m mailer.Mailer) { s.mailer = m }

// SetAccountHooks registers the module that cleans up on account deletion.
func (s *Service) SetAccountHooks(h AccountHooks) { s.hooks = h }

// validEmail accepts a bare address (no display name) of reasonable length.
func validEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email && strings.Contains(email[strings.LastIndex(email, "@"):], ".")
}

func validatePassword(password string) error {
	if len(password) > maxPasswordBytes || len([]rune(password)) < minPasswordRunes {
		return ErrWeakPassword
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// RequestPasswordReset emails a one-time code. It never reveals whether the
// address belongs to an account.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	if s.mailer == nil {
		return errNoMailerAvailable
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, s.resetCodeHash(stored.User.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}

	body := fmt.Sprintf("Votre code de réinitialisation est : %s\n\nIl expire dans %d minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.", code, int(resetCodeTTL.Minutes()))
	if err := s.mailer.Send(ctx, stored.User.Email, "Votre code de réinitialisation", body); err != nil {
		// The caller must not learn whether the account exists, so delivery
		// failures are logged and reported as success.
		log.Printf(`{"event":"password_reset_mail_failed","error":%q}`, err.Error())
	}
	return nil
}

// ResetPassword validates the emailed code, sets the new password and signs
// the user out everywhere.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	reset, err := s.repo.GetActivePasswordReset(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}

	expected := s.resetCodeHash(stored.User.ID, strings.TrimSpace(code))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(reset.CodeHash)) != 1 {
		if err := s.repo.RecordPasswordResetAttempt(ctx, reset.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}

	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, reset.ID, stored.User.ID, hash); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

// ChangePassword verifies the current password, then rotates it and signs out other devices.
func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	if err := s.verifyPassword(ctx, userID, currentPassword); err != nil {
		return err
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	return s.repo.RevokeAllSessions(ctx, userID)
}

// DeleteAccount permanently removes the user and everything they own.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return err
	}
	cleanup := func() {}
	if s.hooks != nil {
		c, err := s.hooks.BeforeDelete(ctx, userID)
		if err != nil {
			return err
		}
		if c != nil {
			cleanup = c
		}
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}
	cleanup()
	return nil
}

// IsSessionActive lets the HTTP middleware reject tokens of revoked or deleted sessions immediately.
func (s *Service) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, userID, sessionID)
}

func (s *Service) verifyPassword(ctx context.Context, userID, password string) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return ErrWrongPassword
	}
	return nil
}

func (s *Service) resetCodeHash(userID, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte(userID + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateNumericCode(digits int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n), nil
}
