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

	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidResetCode    = errors.New("invalid or expired code")
	ErrWeakPassword        = errors.New("password must contain at least one letter and one digit")
	ErrInvalidEmail        = errors.New("a valid email address is required")
)

const (
	accessTokenTTL     = 15 * time.Minute
	refreshTokenTTL    = 30 * 24 * time.Hour
	resetCodeTTL       = 15 * time.Minute
	maxResetAttempts   = 5
	maxResetsPerWindow = 3
	resetWindow        = time.Hour
	bcryptCost         = 12
)

// AccountHooks lets other modules react to account lifecycle events without
// the auth package depending on them.
type AccountHooks interface {
	// AccountDeleted is called after the database rows are gone.
	AccountDeleted(userID string, photoKeys []string)
}

type Service struct {
	repo       Repository
	jwtSecret  []byte
	resetKey   []byte
	mailer     mailer.Mailer
	hooks      AccountHooks
	dummyHash  []byte
	bcryptCost int
}

type Option func(*Service)

// WithBcryptCost lowers the hashing cost (tests only).
func WithBcryptCost(cost int) Option { return func(s *Service) { s.bcryptCost = cost } }

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, hooks AccountHooks, opts ...Option) *Service {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte("password-reset-v1"))
	s := &Service{
		repo:       repo,
		jwtSecret:  jwtSecret,
		resetKey:   mac.Sum(nil),
		mailer:     m,
		hooks:      hooks,
		bcryptCost: bcryptCost,
	}
	for _, opt := range opts {
		opt(s)
	}
	// Used to keep login timing identical for unknown emails.
	s.dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser-password"), s.bcryptCost)
	return s
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized := NormalizeEmail(email)
	if !ValidEmail(normalized) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
	if err := validatePasswordStrength(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
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
	user := toUser(created)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
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
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	user := toUser(stored)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, hashToken(refreshToken))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
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

	next, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(next), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	user := toUser(stored)
	access, err := s.signAccessToken(user.ID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	err := s.repo.RevokeSessionByTokenHash(ctx, hashToken(refreshToken))
	if err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
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
	return toUser(stored), nil
}

// SessionActive is used by the auth middleware so revoked sessions (logout,
// password reset, account deletion) stop authorising access tokens immediately.
func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// RequestPasswordReset never reveals whether the email exists.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	recent, err := s.repo.CountRecentPasswordResets(ctx, stored.ID, time.Now().Add(-resetWindow))
	if err != nil {
		return err
	}
	if recent >= maxResetsPerWindow {
		return nil
	}

	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.ID, s.hashResetCode(stored.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}

	msg := mailer.Message{
		To:      stored.Email,
		Subject: "Votre code de récupération",
		Body: fmt.Sprintf(
			"Bonjour,\n\nVotre code de récupération est : %s\nIl expire dans 15 minutes.\n\nSi vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n",
			code,
		),
	}
	// Delivered asynchronously so response timing does not reveal account existence.
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.mailer.Send(sendCtx, msg); err != nil {
			log.Printf(`{"event":"mail_delivery_failed","error":%q}`, err.Error())
		}
	}()
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	reset, err := s.repo.GetLatestPasswordReset(ctx, stored.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if time.Now().After(reset.ExpiresAt) || reset.Attempts >= maxResetAttempts {
		return ErrInvalidResetCode
	}
	if !hmac.Equal([]byte(reset.CodeHash), []byte(s.hashResetCode(stored.ID, code))) {
		if err := s.repo.IncrementResetAttempts(ctx, reset.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, reset.ID, stored.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

// DeleteAccount requires the current password, then erases the account and its data.
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
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if s.hooks != nil {
		s.hooks.AccountDeleted(userID, keys)
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
	access, err := s.signAccessToken(user.ID, session.ID)
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
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.resetKey)
	mac.Write([]byte(userID))
	mac.Write([]byte{'|'})
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func validatePasswordStrength(password string) error {
	hasLetter, hasDigit := false, false
	for _, r := range password {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127:
			hasLetter = true
		}
	}
	if len(password) < 8 || !hasLetter || !hasDigit {
		return ErrWeakPassword
	}
	return nil
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

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// NormalizeEmail trims and lowercases an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidEmail reports whether an already-normalised address is a plain, valid email.
func ValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" {
		return false
	}
	at := strings.LastIndex(email, "@")
	return at > 0 && strings.Contains(email[at+1:], ".")
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, CreatedAt: stored.CreatedAt}
}
