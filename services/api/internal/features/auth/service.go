package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"example.com/api/internal/platform/authtoken"
	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/realtime"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = apperr.Conflict("email_exists", "an account already exists for this email")
	ErrInvalidCredentials  = apperr.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	ErrInvalidRefreshToken = apperr.New(http.StatusUnauthorized, "invalid_refresh_token", "invalid refresh token")
	ErrUserNotFound        = apperr.New(http.StatusUnauthorized, "unauthorized", "user not found")
	ErrInvalidResetCode    = apperr.New(http.StatusBadRequest, "invalid_reset_code", "invalid or expired code")
	ErrWeakPassword        = apperr.Validation("password must contain between 8 and 72 characters, including a letter and a digit")
	ErrResetUnavailable    = apperr.New(http.StatusServiceUnavailable, "reset_unavailable", "password reset is not configured on this server")
	ErrAccountSuspended    = apperr.New(http.StatusForbidden, "account_suspended", "this account has been suspended")
)

const (
	refreshTokenTTL      = 30 * 24 * time.Hour
	resetCodeTTL         = 30 * time.Minute
	maxResetCodeAttempts = 5
	resetResendInterval  = time.Minute
)

// AccountCleaner releases resources owned by a user that the database cascade
// cannot remove itself (files on disk). It is called before the user row is
// deleted and returns a function that finalizes the cleanup afterwards.
type AccountCleaner interface {
	PrepareAccountDeletion(ctx context.Context, userID string) (finalize func(context.Context), err error)
}

type Service struct {
	disconnector realtime.Disconnector
	repo         Repository
	tokens       *authtoken.Manager
	mailer       mailer.Mailer
	cleaners     []AccountCleaner
	// dummyHash equalizes login timing for unknown emails.
	dummyHash []byte
}

func NewService(repo Repository, tokens *authtoken.Manager, mail mailer.Mailer, cleaners ...AccountCleaner) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equalizer-password"), bcrypt.DefaultCost)
	return &Service{repo: repo, tokens: tokens, mailer: mail, cleaners: cleaners, dummyHash: dummy}
}

// SetDisconnector lets the service close live realtime connections when an
// account loses access (password reset, deletion).
func (s *Service) SetDisconnector(d realtime.Disconnector) {
	s.disconnector = d
}

func (s *Service) disconnect(ctx context.Context, userID string) {
	if s.disconnector != nil {
		s.disconnector.DisconnectUsers(ctx, []string{userID})
	}
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}

	user, err := s.repo.CreateUser(ctx, NormalizeEmail(email), string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}

	tokens, err := s.issueSessionTokens(ctx, user.ID, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	if stored.User.Suspended {
		return Tokens{}, User{}, ErrAccountSuspended
	}

	tokens, err := s.issueSessionTokens(ctx, stored.User.ID, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, stored.User, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	currentHash := hashSecret(strings.TrimSpace(refreshToken))
	if currentHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	sessionID, userID, err := s.repo.GetActiveSessionByTokenHash(ctx, currentHash)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrUserNotFound
		}
		return Tokens{}, User{}, err
	}
	if stored.User.Suspended {
		return Tokens{}, User{}, ErrAccountSuspended
	}

	nextRefresh, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	err = s.repo.RotateSessionToken(ctx, sessionID, currentHash, hashSecret(nextRefresh), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	accessToken, err := s.tokens.SignAccess(userID, sessionID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: nextRefresh}, stored.User, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashSecret(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return ErrInvalidRefreshToken
	}
	return s.repo.RevokeSessionByTokenHash(ctx, tokenHash)
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return stored.User, nil
}

// SessionActive is used by the auth middleware on every request.
func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.SessionActive(ctx, sessionID, userID)
}

// ForgotPassword emails a 6-digit reset code. It never reveals whether the
// email is registered: unknown emails succeed silently.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	if s.mailer == nil {
		return ErrResetUnavailable
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}

	// At most one email per minute per account, whatever the caller's IP.
	existing, err := s.repo.GetResetCode(ctx, stored.User.ID)
	if err == nil && time.Since(existing.CreatedAt) < resetResendInterval {
		return nil
	}
	if err != nil && !errors.Is(err, ErrRepositoryNotFound) {
		return err
	}

	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.UpsertResetCode(ctx, stored.User.ID, hashSecret(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}

	body := fmt.Sprintf("Bonjour,\n\nVoici votre code de réinitialisation : %s\n\nIl expire dans 30 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez cet email.\n", code)
	if err := s.mailer.Send(ctx, stored.User.Email, "Votre code de réinitialisation", body); err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","error":%q}`, err.Error())
		return apperr.New(http.StatusServiceUnavailable, "mail_failed", "could not send the reset email, please retry later")
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	// The attempt is counted atomically before comparing, so concurrent
	// guesses cannot exceed maxResetCodeAttempts.
	codeHash, err := s.repo.ConsumeResetAttempt(ctx, stored.User.ID, maxResetCodeAttempts)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if subtle.ConstantTimeCompare([]byte(codeHash), []byte(hashSecret(strings.TrimSpace(code)))) != 1 {
		return ErrInvalidResetCode
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.ResetPassword(ctx, stored.User.ID, string(hash)); err != nil {
		return err
	}
	s.disconnect(ctx, stored.User.ID)
	return nil
}

// DeleteAccount permanently removes the account after password confirmation.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return apperr.New(http.StatusForbidden, "invalid_password", "password is incorrect")
	}

	finalizers := make([]func(context.Context), 0, len(s.cleaners))
	for _, cleaner := range s.cleaners {
		finalize, err := cleaner.PrepareAccountDeletion(ctx, userID)
		if err != nil {
			return err
		}
		finalizers = append(finalizers, finalize)
	}

	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}
	s.disconnect(ctx, userID)
	for _, finalize := range finalizers {
		finalize(ctx)
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, userID, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	sessionID, err := s.repo.CreateSession(ctx, userID, hashSecret(refreshToken), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := s.tokens.SignAccess(userID, sessionID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// ValidatePassword enforces length (bcrypt only uses 72 bytes) and a minimal
// mix of letters and digits.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return ErrWeakPassword
	}
	hasLetter, hasDigit := false, false
	for _, r := range password {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127:
			hasLetter = true
		}
	}
	if !hasLetter || !hasDigit {
		return ErrWeakPassword
	}
	return nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomDigits(n int) (string, error) {
	var b strings.Builder
	for i := 0; i < n; i++ {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + d.Int64()))
	}
	return b.String(), nil
}

func hashSecret(secret string) string {
	if secret == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
