package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
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

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/tokens"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrWeakPassword        = errors.New("weak password")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidResetCode    = errors.New("invalid reset code")
	ErrUserNotFound        = errors.New("user not found")
)

const (
	refreshTokenTTL  = 30 * 24 * time.Hour
	resetCodeTTL     = 15 * time.Minute
	maxResetAttempts = 5
	bcryptCost       = 12
)

// AccountDataCleaner removes data living outside Postgres (photo files) and
// live connections when an account is deleted.
type AccountDataCleaner interface {
	StorageKeysForUser(ctx context.Context, userID string) ([]string, error)
	DeleteObjects(keys []string)
}

type Disconnector interface {
	DisconnectUser(userID string)
}

type Service struct {
	repo       Repository
	tokens     *tokens.Manager
	resetKey   []byte
	mailer     mailer.Mailer
	cleaner    AccountDataCleaner
	disconnect Disconnector
	dummyHash  []byte
}

func NewService(repo Repository, tm *tokens.Manager, appSecret []byte, m mailer.Mailer, cleaner AccountDataCleaner, d Disconnector) *Service {
	mac := hmac.New(sha256.New, appSecret)
	mac.Write([]byte("password-reset-v1"))
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcryptCost)
	return &Service{repo: repo, tokens: tm, resetKey: mac.Sum(nil), mailer: m, cleaner: cleaner, disconnect: d, dummyHash: dummy}
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidatePassword enforces 8-72 bytes with at least one letter and one digit
// (72 bytes is the bcrypt input limit).
func ValidatePassword(p string) error {
	if len(p) < 8 || len(p) > 72 {
		return ErrWeakPassword
	}
	var letter, digit bool
	for _, r := range p {
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

func validEmail(email string) bool {
	if len(email) > 254 || strings.ContainsAny(email, " \r\n") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email && strings.Contains(email[strings.LastIndex(email, "@"):], ".")
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	email = NormalizeEmail(email)
	if !validEmail(email) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	user, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	t, err := s.issue(ctx, user.ID, userAgent, ip)
	return t, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			// Equalize timing so response time does not reveal registered e-mails.
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	t, err := s.issue(ctx, stored.ID, userAgent, ip)
	return t, stored.User, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	current := hashToken(refreshToken)
	if current == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	next, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	session, err := s.repo.RotateSession(ctx, current, hashToken(next), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	access, err := s.tokens.SignAccess(session.UserID, session.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, stored.User, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	h := hashToken(refreshToken)
	if h == "" {
		return ErrInvalidRefreshToken
	}
	return s.repo.RevokeSession(ctx, h)
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return User{}, ErrUserNotFound
	}
	return stored.User, err
}

func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// RequestPasswordReset always succeeds from the caller's point of view so the
// endpoint cannot be used to discover which e-mails are registered.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = NormalizeEmail(email)
	stored, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	code, err := randomCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.ID, s.hashResetCode(stored.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Bonjour,\n\nVoici ton code de réinitialisation : %s\nIl expire dans 15 minutes.\n\nSi tu n'es pas à l'origine de cette demande, ignore ce message.\n", code)
	if err := s.mailer.Send(ctx, stored.Email, "Ton code de réinitialisation", body); err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","user_id":%q,"error":%q}`, stored.ID, err.Error())
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	reset, err := s.repo.GetActivePasswordReset(ctx, stored.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if reset.Attempts >= maxResetAttempts {
		return ErrInvalidResetCode
	}
	if !hmac.Equal([]byte(reset.CodeHash), []byte(s.hashResetCode(stored.ID, strings.TrimSpace(code)))) {
		if err := s.repo.IncrementResetAttempts(ctx, reset.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, reset.ID, stored.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	s.disconnect.DisconnectUser(stored.ID)
	return nil
}

// DeleteAccount permanently removes the user after re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	keys, err := s.cleaner.StorageKeysForUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}
	s.cleaner.DeleteObjects(keys)
	s.disconnect.DisconnectUser(userID)
	return nil
}

func (s *Service) issue(ctx context.Context, userID, userAgent, ip string) (Tokens, error) {
	refresh, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, userID, hashToken(refresh), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := s.tokens.SignAccess(userID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.resetKey)
	mac.Write([]byte(userID))
	mac.Write([]byte{0})
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
