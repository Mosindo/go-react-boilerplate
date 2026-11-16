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
	"example.com/api/internal/platform/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrWeakPassword        = errors.New("password must contain at least 8 characters, including a letter and a digit")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrInvalidEmail        = errors.New("invalid email address")
)

const (
	accessTokenTTL       = 15 * time.Minute
	refreshTokenTTL      = 30 * 24 * time.Hour
	resetCodeTTL         = 30 * time.Minute
	resetMaxAttempts     = 5
	resetMaxPerHour      = 3
	bcryptCost           = 12
	passwordResetSubject = "Ton code de réinitialisation"
)

// AccountObserver is notified after an account is deleted (e.g. to close its
// realtime connections).
type AccountObserver interface {
	DisconnectUser(userID string)
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	resetKey  []byte
	mailer    mailer.Mailer
	storage   storage.Storage
	observer  AccountObserver
	// dummyHash keeps login timing constant for unknown emails.
	dummyHash []byte
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, store storage.Storage, observer AccountObserver) *Service {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte("password-reset-code-v1"))
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		resetKey:  mac.Sum(nil),
		mailer:    m,
		storage:   store,
		observer:  observer,
		dummyHash: dummy,
	}
}

// ValidatePassword enforces the password policy.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
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

// ValidateEmail checks an already-normalized email address.
func ValidateEmail(email string) error {
	if len(email) < 3 || len(email) > 254 || strings.ContainsAny(email, " \t\r\n<>,;\"") {
		return ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ErrInvalidEmail
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || !strings.Contains(email[at+1:], ".") || strings.HasSuffix(email, ".") {
		return ErrInvalidEmail
	}
	return nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return Tokens{}, User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	created, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	user := toUser(created)
	tokens, err := s.issueSessionTokens(ctx, user.ID, userAgent, ip)
	return tokens, user, err
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
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	user := toUser(stored)
	tokens, err := s.issueSessionTokens(ctx, user.ID, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	currentHash := hashToken(refreshToken)
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
	next, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, currentHash, hashToken(next), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByID(ctx, rotated.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrUserNotFound
		}
		return Tokens{}, User{}, err
	}
	access, err := s.signAccessToken(rotated.UserID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, toUser(stored), nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	h := hashToken(refreshToken)
	if h == "" {
		return ErrInvalidRefreshToken
	}
	return s.repo.RevokeSessionByTokenHash(ctx, h)
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

func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// RequestPasswordReset emails a 6-digit code. It never reveals whether the
// email exists: callers always answer 202.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	recent, err := s.repo.CountRecentPasswordResets(ctx, stored.ID, time.Now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= resetMaxPerHour {
		return nil
	}
	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.ID, s.hashResetCode(stored.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Voici ton code de réinitialisation : %s\n\nIl expire dans 30 minutes. Si tu n'es pas à l'origine de cette demande, ignore ce message.", code)
	if err := s.mailer.Send(ctx, stored.Email, passwordResetSubject, body); err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","error":%q}`, err.Error())
	}
	return nil
}

// ResetPassword checks the code, sets the new password and revokes every
// session so a compromised device is logged out.
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
	// The attempt is reserved atomically before comparing, so parallel
	// guesses can never exceed resetMaxAttempts.
	reset, err := s.repo.ReserveResetAttempt(ctx, stored.ID, resetMaxAttempts)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if !hmac.Equal([]byte(reset.CodeHash), []byte(s.hashResetCode(stored.ID, code))) {
		return ErrInvalidResetCode
	}
	consumed, err := s.repo.ConsumePasswordReset(ctx, reset.ID)
	if err != nil {
		return err
	}
	if !consumed {
		return ErrInvalidResetCode
	}
	return s.setPassword(ctx, stored.ID, newPassword)
}

func (s *Service) ChangePassword(ctx context.Context, userID, current, next string) error {
	if err := ValidatePassword(next); err != nil {
		return err
	}
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return ErrInvalidCredentials
	}
	return s.setPassword(ctx, userID, next)
}

func (s *Service) setPassword(ctx context.Context, userID, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	if err := s.repo.RevokeAllSessions(ctx, userID); err != nil {
		return err
	}
	if s.observer != nil {
		s.observer.DisconnectUser(userID)
	}
	return nil
}

// DeleteAccount permanently deletes the user and all associated data, after
// re-confirming the password.
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
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := s.storage.Delete(ctx, key); err != nil {
			log.Printf(`{"event":"account_delete_file_failed","key":%q,"error":%q}`, key, err.Error())
		}
	}
	if s.observer != nil {
		s.observer.DisconnectUser(userID)
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, userID, userAgent, ip string) (Tokens, error) {
	refresh, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, userID, hashToken(refresh), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := s.signAccessToken(userID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *Service) signAccessToken(userID, sessionID string) (string, error) {
	now := time.Now()
	claims := AccessTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.resetKey)
	mac.Write([]byte(userID + "|" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomDigits(n int) (string, error) {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		sb.WriteByte(byte('0' + d.Int64()))
	}
	return sb.String(), nil
}

func hashToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func toUser(s StoredUser) User {
	return User{ID: s.ID, Email: s.Email, HasProfile: s.HasProfile, CreatedAt: s.CreatedAt}
}
