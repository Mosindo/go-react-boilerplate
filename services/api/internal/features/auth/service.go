package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/token"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrWeakPassword        = errors.New("password must be 8 to 72 bytes")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidResetCode    = errors.New("invalid or expired code")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetCodeTTL    = 30 * time.Minute
	maxResetTries   = 5
	minPasswordLen  = 8
	maxPasswordLen  = 72 // bcrypt limit
)

// dummyHash lets Login spend the same time for unknown emails as for known ones.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

// SessionInvalidator drops cached session state after a revocation.
type SessionInvalidator interface {
	ForgetSession(sessionID string)
	ForgetUser(userID string)
}

type Service struct {
	repo        Repository
	jwtSecret   []byte
	mailer      mailer.Mailer
	files       storage.Store
	invalidator SessionInvalidator
	cost        int
}

type Option func(*Service)

func WithInvalidator(i SessionInvalidator) Option { return func(s *Service) { s.invalidator = i } }

// WithBcryptCost lowers the work factor; intended for tests only.
func WithBcryptCost(cost int) Option { return func(s *Service) { s.cost = cost } }

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, files storage.Store, opts ...Option) *Service {
	s := &Service{repo: repo, jwtSecret: jwtSecret, mailer: m, files: files, cost: bcrypt.DefaultCost}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	user := toUser(stored)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	user := toUser(stored)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, hashToken(strings.TrimSpace(refreshToken)))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
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
	next, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(next), userAgent, ip, time.Now().Add(refreshTokenTTL)); err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	access, err := token.Sign(s.jwtSecret, stored.ID, session.ID, token.TypeAccess, accessTokenTTL)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, toUser(stored), nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.repo.RevokeSessionByTokenHash(ctx, hashToken(strings.TrimSpace(refreshToken)))
	if err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	if err == nil && s.invalidator != nil {
		s.invalidator.ForgetSession(session.ID)
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return User{}, ErrUserNotFound
	}
	return toUser(stored), err
}

// ForgotPassword never reveals whether the email exists.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return nil
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	code, err := randomResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.ID, hashToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Votre code de réinitialisation : %s\n\nIl expire dans 30 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.", code)
	return s.mailer.Send(ctx, stored.Email, "Réinitialisation de votre mot de passe", body)
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	normalized, err := NormalizeEmail(email)
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
	reset, err := s.repo.GetActivePasswordReset(ctx, stored.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if reset.Attempts >= maxResetTries {
		return ErrInvalidResetCode
	}
	candidate := hashToken(strings.ToUpper(strings.TrimSpace(code)))
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(reset.CodeHash)) != 1 {
		_ = s.repo.RecordResetAttempt(ctx, reset.ID)
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.cost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, reset.ID, stored.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if s.invalidator != nil {
		s.invalidator.ForgetUser(stored.ID)
	}
	return nil
}

// ChangePassword keeps the current session and revokes all the others.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := ValidatePassword(next); err != nil {
		return err
	}
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), s.cost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	if err := s.repo.RevokeUserSessions(ctx, userID, sessionID); err != nil {
		return err
	}
	if s.invalidator != nil {
		s.invalidator.ForgetUser(userID)
	}
	return nil
}

// DeleteAccount erases the user and everything attached to it (profile, photos, swipes,
// matches, messages, blocks, reports, notifications) after re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	if s.invalidator != nil {
		s.invalidator.ForgetUser(userID)
	}
	for _, key := range keys {
		_ = s.files.Delete(ctx, key) // best effort: the DB rows are already gone
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ip string) (Tokens, error) {
	refresh, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashToken(refresh), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := token.Sign(s.jwtSecret, user.ID, session.ID, token.TypeAccess, accessTokenTTL)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

// NormalizeEmail trims, lowercases and validates the address.
func NormalizeEmail(email string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(email))
	if trimmed == "" || len(trimmed) > 254 {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed || addr.Name != "" || !strings.Contains(trimmed[strings.LastIndex(trimmed, "@"):], ".") {
		return "", ErrInvalidEmail
	}
	return trimmed, nil
}

func ValidatePassword(p string) error {
	if len(p) < minPasswordLen || len(p) > maxPasswordLen || !utf8.ValidString(p) {
		return ErrWeakPassword
	}
	return nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// randomResetCode returns 10 base32 characters (50 bits), easy to type from an email.
func randomResetCode() (string, error) {
	b := make([]byte, 7)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(b)[:10], nil
}

func hashToken(t string) string {
	if t == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func toUser(s StoredUser) User { return User{ID: s.ID, Email: s.Email, CreatedAt: s.CreatedAt} }
