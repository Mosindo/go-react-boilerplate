package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/token"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = apperr.New(http.StatusConflict, "email_exists", "an account already exists with this email")
	ErrInvalidEmail        = apperr.Validation("invalid_email", "invalid email address")
	ErrWeakPassword        = apperr.Validation("weak_password", "password must be 8 to 72 bytes long and contain a letter and a digit")
	ErrInvalidCredentials  = apperr.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	ErrInvalidRefreshToken = apperr.New(http.StatusUnauthorized, "invalid_refresh_token", "session expired, please sign in again")
	ErrInvalidResetCode    = apperr.Validation("invalid_reset_code", "this code is invalid or expired")
	ErrWrongPassword       = apperr.New(http.StatusForbidden, "wrong_password", "current password is incorrect")
	ErrUserNotFound        = apperr.New(http.StatusUnauthorized, "user_not_found", "account not found")
)

const (
	AccessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetCodeTTL    = 30 * time.Minute
	bcryptCost      = bcrypt.DefaultCost
)

// dummyHash is compared against when an email is unknown so that login
// timing does not reveal which emails are registered.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-password-1"), bcryptCost)

type Service struct {
	repo    Repository
	tokens  *token.Manager
	mailer  mailer.Mailer
	storage storage.Storage
	now     func() time.Time
}

func NewService(repo Repository, tokens *token.Manager, m mailer.Mailer, s storage.Storage) *Service {
	return &Service{repo: repo, tokens: tokens, mailer: m, storage: s, now: time.Now}
}

// IsSessionActive implements middleware.SessionChecker.
func (s *Service) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, userID, sessionID)
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := NormalizeEmail(email)
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
	if errors.Is(err, errRepoEmailExists) {
		return Tokens{}, User{}, ErrEmailExists
	}
	if err != nil {
		return Tokens{}, User{}, err
	}
	tokens, err := s.issue(ctx, user.ID, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if errors.Is(err, errRepoNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	if err != nil {
		return Tokens{}, User{}, err
	}
	// Always run the comparison so demo accounts are not distinguishable by timing.
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil || stored.IsDemo {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	tokens, err := s.issue(ctx, stored.ID, userAgent, ip)
	return tokens, stored.User, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	oldHash := hashSecret(strings.TrimSpace(refreshToken))
	if oldHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	next, err := randomToken(32)
	if err != nil {
		return Tokens{}, User{}, err
	}
	sessionID, userID, err := s.repo.RotateSession(ctx, oldHash, hashSecret(next), userAgent, ip, s.now().Add(refreshTokenTTL))
	if errors.Is(err, errRepoNotFound) {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, errRepoNotFound) {
		return Tokens{}, User{}, ErrUserNotFound
	}
	if err != nil {
		return Tokens{}, User{}, err
	}
	access, err := s.tokens.Sign(userID, sessionID, s.now())
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, stored.User, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	hash := hashSecret(strings.TrimSpace(refreshToken))
	if hash == "" {
		return nil
	}
	return s.repo.RevokeSessionByHash(ctx, hash)
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, errRepoNotFound) {
		return User{}, ErrUserNotFound
	}
	return stored.User, err
}

// ForgotPassword sends a one-time recovery code. It never reveals whether the
// email is registered.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return nil
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if errors.Is(err, errRepoNotFound) || (err == nil && stored.IsDemo) {
		return nil
	}
	if err != nil {
		return err
	}
	code, err := newResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreateResetToken(ctx, stored.ID, hashSecret(code), s.now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf(
		"Bonjour,\n\nVoici votre code de réinitialisation Lueur : %s\n\nIl est valable 30 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n",
		formatResetCode(code),
	)
	// Sent asynchronously so the response time does not reveal registered emails.
	go func(to string) {
		sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if err := s.mailer.Send(sendCtx, to, "Votre code de réinitialisation", body); err != nil {
			log.Printf(`{"event":"reset_mail_failed","error":%q}`, err.Error())
		}
	}(stored.Email)
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, code, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	normalized := normalizeResetCode(code)
	if len(normalized) != resetCodeLength {
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return err
	}
	err = s.repo.ConsumeResetToken(ctx, hashSecret(normalized), string(hash))
	if errors.Is(err, errRepoNotFound) {
		return ErrInvalidResetCode
	}
	return err
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, errRepoNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return ErrWrongPassword
	}
	if err := ValidatePassword(next); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcryptCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

// DeleteAccount permanently erases the account, its data and its photo files.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, errRepoNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return ErrWrongPassword
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if errors.Is(err, errRepoNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := s.storage.Delete(ctx, key); err != nil {
			log.Printf(`{"event":"photo_delete_failed","key":%q,"error":%q}`, key, err.Error())
		}
	}
	return nil
}

func (s *Service) issue(ctx context.Context, userID, userAgent, ip string) (Tokens, error) {
	refresh, err := randomToken(32)
	if err != nil {
		return Tokens{}, err
	}
	sessionID, err := s.repo.CreateSession(ctx, userID, hashSecret(refresh), userAgent, ip, s.now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := s.tokens.Sign(userID, sessionID, s.now())
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

// NormalizeEmail trims, lower-cases and validates an email address.
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if len(email) < 3 || len(email) > 254 {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || !strings.Contains(email[at+1:], ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// ValidatePassword enforces the password policy.
func ValidatePassword(password string) error {
	// bcrypt only accepts up to 72 bytes.
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

const resetCodeLength = 10

// Crockford-like alphabet without ambiguous characters (0/O, 1/I).
var resetEncoding = base32.NewEncoding("ABCDEFGHJKLMNPQRSTUVWXYZ23456789").WithPadding(base32.NoPadding)

func newResetCode() (string, error) {
	buf := make([]byte, 7)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return resetEncoding.EncodeToString(buf)[:resetCodeLength], nil
}

func formatResetCode(code string) string {
	return code[:5] + "-" + code[5:]
}

func normalizeResetCode(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(raw) {
		if r == '-' || unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashSecret(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
