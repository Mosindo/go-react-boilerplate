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

	"example.com/api/internal/platform/authtoken"
	"example.com/api/internal/platform/mailer"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrWeakPassword        = errors.New("password must be 8-128 characters with at least one letter and one digit")
	ErrInvalidEmail        = errors.New("invalid email address")
)

const (
	refreshTokenTTL       = 30 * 24 * time.Hour
	resetCodeTTL          = 15 * time.Minute
	maxResetAttempts      = 5
	maxResetRequestsPerHr = 3
	bcryptCost            = 12
)

// AccountHooks lets the composition root clean up resources owned by other
// modules (stored photo files, open realtime connections) after deletion.
type AccountHooks interface {
	AccountDeleted(ctx context.Context, userID string, photoKeys []string)
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	hooks     AccountHooks
	dummyHash []byte
	now       func() time.Time
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, hooks AccountHooks) *Service {
	// Used to keep login timing constant when the email does not exist.
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equalizer-password"), bcryptCost)
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: m, hooks: hooks, dummyHash: dummy, now: time.Now}
}

// ValidatePassword enforces the password policy shared by every flow.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
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

// ValidateEmail normalizes and validates an email address (bare address only,
// no display name).
func ValidateEmail(email string) (string, error) {
	normalized := NormalizeEmail(email)
	if len(normalized) < 3 || len(normalized) > 254 {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized || !strings.Contains(normalized[strings.LastIndex(normalized, "@"):], ".") {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := ValidateEmail(email)
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
	created, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	tokens, err := s.issueSession(ctx, created.ID, userAgent, ip)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, toUser(created), nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
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
	tokens, err := s.issueSession(ctx, stored.ID, userAgent, ip)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, toUser(stored), nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	currentHash := hashToken(refreshToken)
	if currentHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	next, err := generateToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	session, err := s.repo.RotateSession(ctx, currentHash, hashToken(next), userAgent, ip, s.now().Add(refreshTokenTTL))
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
	access, err := authtoken.Sign(s.jwtSecret, stored.ID, session.ID, s.now())
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, toUser(stored), nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	if tokenHash == "" {
		return nil
	}
	return s.repo.RevokeSessionByTokenHash(ctx, tokenHash)
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return toUser(stored), nil
}

// ForgotPassword always succeeds from the caller's perspective so it cannot
// be used to discover which emails have an account.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	recent, err := s.repo.CountRecentResets(ctx, stored.ID, s.now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= maxResetRequestsPerHr {
		return nil
	}
	code, err := generateNumericCode(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.ID, s.hashResetCode(stored.ID, code), s.now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Bonjour,\n\nVotre code de réinitialisation Lueur est : %s\n\nIl expire dans 15 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n", code)
	if err := s.mailer.Send(ctx, stored.Email, "Votre code de réinitialisation Lueur", body); err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","error":%q}`, err.Error())
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
	if err := s.repo.ConsumeResetAndSetPassword(ctx, reset.ID, stored.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, currentPassword, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	if err := s.verifyPassword(ctx, userID, currentPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordAndRevokeSessions(ctx, userID, string(hash), sessionID)
}

func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return err
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if s.hooks != nil {
		s.hooks.AccountDeleted(ctx, userID, keys)
	}
	return nil
}

func (s *Service) verifyPassword(ctx context.Context, userID, password string) error {
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
	return nil
}

func (s *Service) issueSession(ctx context.Context, userID, userAgent, ip string) (Tokens, error) {
	refresh, err := generateToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, userID, hashToken(refresh), userAgent, ip, s.now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := authtoken.Sign(s.jwtSecret, userID, session.ID, s.now())
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte("lueur/password-reset/v1|" + userID + "|" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateNumericCode(digits int) (string, error) {
	var b strings.Builder
	for i := 0; i < digits; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + n.Int64()))
	}
	return b.String(), nil
}

func hashToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func toUser(s StoredUser) User {
	return User{ID: s.ID, Email: s.Email, CreatedAt: s.CreatedAt}
}
