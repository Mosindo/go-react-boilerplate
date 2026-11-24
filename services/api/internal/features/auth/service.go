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
	"net/mail"
	"strings"
	"time"
	"unicode"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/tokens"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = apperr.Conflict("email_taken", "an account already exists for this email")
	ErrInvalidCredentials  = apperr.Unauthorized("invalid_credentials", "incorrect email or password")
	ErrInvalidRefreshToken = apperr.Unauthorized("invalid_refresh_token", "session expired, please sign in again")
	ErrUserNotFound        = apperr.Unauthorized("user_not_found", "account not found")
	ErrInvalidEmail        = apperr.BadRequest("invalid_email", "please enter a valid email address")
	ErrWeakPassword        = apperr.BadRequest("weak_password", "password must contain at least 8 characters, one letter and one digit")
	ErrInvalidResetCode    = apperr.BadRequest("invalid_reset_code", "this code is invalid or expired")
	ErrTooManyResets       = apperr.TooManyRequests("too_many_resets", "too many reset requests, retry later")
)

const (
	AccessTokenTTL      = 15 * time.Minute
	refreshTokenTTL     = 30 * 24 * time.Hour
	resetCodeTTL        = 30 * time.Minute
	maxResetAttempts    = 5
	maxResetsPerHour    = 3
	resetCodeDigits     = 8
	bcryptCost          = 12
	dummyPasswordSecret = "lueur-timing-equalizer"
)

// AccountEraser cleans up data stored outside Postgres (photo files) and live
// connections when an account is deleted.
type AccountEraser interface {
	StorageKeysForUser(ctx context.Context, userID string) ([]string, error)
	PurgeObjects(ctx context.Context, keys []string)
}

type Disconnector interface {
	DisconnectUser(userID string)
	DisconnectSession(userID, sessionID string)
}

type Service struct {
	repo       Repository
	tokens     *tokens.Manager
	mailer     mailer.Mailer
	eraser     AccountEraser
	disconnect Disconnector
	dummyHash  []byte
}

func NewService(repo Repository, manager *tokens.Manager, mail mailer.Mailer, eraser AccountEraser, disconnect Disconnector) *Service {
	dummy, err := bcrypt.GenerateFromPassword([]byte(dummyPasswordSecret), bcryptCost)
	if err != nil {
		panic(fmt.Sprintf("generate dummy hash: %v", err))
	}
	return &Service{repo: repo, tokens: manager, mailer: mail, eraser: eraser, disconnect: disconnect, dummyHash: dummy}
}

// ValidatePassword enforces the minimal password policy.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return ErrWeakPassword
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
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

// ParseEmail normalises then validates an email address.
func ParseEmail(raw string) (string, error) {
	email := NormalizeEmail(raw)
	if len(email) < 3 || len(email) > 254 {
		return "", ErrInvalidEmail
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || !strings.Contains(email[at+1:], ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := ParseEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	user, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	issued, err := s.issueSession(ctx, user.ID, userAgent, ip)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return issued, user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := ParseEmail(email)
	if err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			// Equalise timing so response latency does not reveal accounts.
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	issued, err := s.issueSession(ctx, stored.ID, userAgent, ip)
	if err != nil {
		return Tokens{}, User{}, err
	}
	_ = s.repo.TouchUser(ctx, stored.ID)
	return issued, stored.User, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	currentHash := hashSecret(strings.TrimSpace(refreshToken))
	if currentHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, currentHash)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrUserNotFound
		}
		return Tokens{}, User{}, err
	}
	nextRefresh, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := s.repo.RotateSessionToken(ctx, session.ID, currentHash, hashSecret(nextRefresh), userAgent, ip, time.Now().Add(refreshTokenTTL)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	access, err := s.tokens.Sign(stored.ID, session.ID, tokens.AudienceAccess, AccessTokenTTL)
	if err != nil {
		return Tokens{}, User{}, err
	}
	_ = s.repo.TouchUser(ctx, stored.ID)
	return Tokens{AccessToken: access, RefreshToken: nextRefresh}, stored.User, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashSecret(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return ErrInvalidRefreshToken
	}
	session, err := s.repo.RevokeSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	if s.disconnect != nil {
		s.disconnect.DisconnectSession(session.UserID, session.ID)
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return stored.User, nil
}

// IsSessionActive is used by the auth middleware.
func (s *Service) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, userID, sessionID)
}

// ForgotPassword emails a one-time code. It never reveals whether the email
// belongs to an account.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized, err := ParseEmail(email)
	if err != nil {
		return err
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	recent, err := s.repo.CountRecentResets(ctx, stored.ID, time.Now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= maxResetsPerHour {
		return nil
	}
	code, err := randomDigits(resetCodeDigits)
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.ID, hashResetCode(stored.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := "Bonjour,\n\nVoici ton code de réinitialisation Lueur : " + code +
		"\n\nIl expire dans 30 minutes. Si tu n'es pas à l'origine de cette demande, ignore ce message."
	if err := s.mailer.Send(ctx, stored.Email, "Ton code Lueur", body); err != nil {
		log.Printf(`{"event":"password_reset_email_failed","error":%q}`, err.Error())
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	normalized, err := ParseEmail(email)
	if err != nil {
		return ErrInvalidResetCode
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	reset, err := s.repo.GetLatestActiveReset(ctx, stored.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	attempts, err := s.repo.IncrementResetAttempts(ctx, reset.ID)
	if err != nil {
		return err
	}
	if attempts > maxResetAttempts {
		return ErrInvalidResetCode
	}
	expected := hashResetCode(stored.ID, strings.TrimSpace(code))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(reset.CodeHash)) != 1 {
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, stored.ID, reset.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if s.disconnect != nil {
		s.disconnect.DisconnectUser(stored.ID)
	}
	return nil
}

// DeleteAccount permanently erases the account after re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	var keys []string
	if s.eraser != nil {
		keys, err = s.eraser.StorageKeysForUser(ctx, userID)
		if err != nil {
			return err
		}
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if s.eraser != nil {
		s.eraser.PurgeObjects(ctx, keys)
	}
	if s.disconnect != nil {
		s.disconnect.DisconnectUser(userID)
	}
	return nil
}

func (s *Service) issueSession(ctx context.Context, userID, userAgent, ip string) (Tokens, error) {
	refresh, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, userID, hashSecret(refresh), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := s.tokens.Sign(userID, session.ID, tokens.AudienceAccess, AccessTokenTTL)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
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

func hashResetCode(userID, code string) string {
	return hashSecret(userID + ":" + code)
}
