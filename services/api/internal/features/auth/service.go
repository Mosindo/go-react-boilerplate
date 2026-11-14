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
	"math/big"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
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
	accessTokenTTL     = 15 * time.Minute
	refreshTokenTTL    = 30 * 24 * time.Hour
	resetCodeTTL       = 30 * time.Minute
	maxResetAttempts   = 5
	resetCodeDigits    = 6
	minPasswordLength  = 8
	maxPasswordBytes   = 72 // bcrypt limit
	accessTokenIssuer  = "lueur-api"
	accessTokenSubject = "access"
)

// AccountHooks lets other features react to account lifecycle events
// without auth importing them (e.g. purge photo files, close sockets).
type AccountHooks struct {
	// BeforeDelete returns a function to run once the user row is gone.
	BeforeDelete func(ctx context.Context, userID string) (after func(), err error)
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	hooks     AccountHooks
	now       func() time.Time
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, hooks AccountHooks) *Service {
	if m == nil {
		m = mailer.LogMailer{}
	}
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: m, hooks: hooks, now: time.Now}
}

// NormalizeEmail trims, lowercases and validates an e-mail address.
func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" || len(normalized) > 254 {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized || !strings.Contains(normalized[strings.LastIndex(normalized, "@")+1:], ".") {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

// ValidatePassword enforces 8-72 bytes with at least one letter and one digit.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength || len(password) > maxPasswordBytes {
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

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	user, err := s.repo.CreateUser(ctx, normalizedEmail, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

// dummyHash keeps login timing similar for unknown e-mails.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser-not-a-password-1"), bcrypt.DefaultCost)

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	tokens, err := s.issueSessionTokens(ctx, stored.User, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, stored.User, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	currentHash := hashToken(strings.TrimSpace(refreshToken))
	if currentHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	nextSecret, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	sessionID, userID, err := s.repo.RotateSession(ctx, currentHash, hashToken(nextSecret), userAgent, ipAddress, s.now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	accessToken, err := s.signAccessToken(stored.User.ID, sessionID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: nextSecret}, stored.User, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(strings.TrimSpace(refreshToken))
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

// SessionActive is used by the auth middleware for immediate revocation.
func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// RequestPasswordReset sends a 6-digit code. It never reveals whether the
// e-mail exists: unknown addresses succeed silently.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return ErrInvalidEmail
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	code, err := generateNumericCode(resetCodeDigits)
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashResetCode(stored.User.ID, code), s.now().Add(resetCodeTTL)); err != nil {
		return err
	}
	return s.mailer.Send(ctx, mailer.Message{
		To:      stored.User.Email,
		Subject: "Votre code de réinitialisation Lueur",
		Body: fmt.Sprintf(
			"Bonjour,\n\nVotre code de réinitialisation est : %s\nIl expire dans 30 minutes.\n\nSi vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n",
			code,
		),
	})
}

// ResetPassword verifies the code (max 5 attempts per code), sets the new
// password and revokes every existing session.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return ErrInvalidResetCode
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizedEmail)
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
	if reset.Attempts >= maxResetAttempts {
		return ErrInvalidResetCode
	}
	candidate := hashResetCode(stored.User.ID, strings.TrimSpace(code))
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(reset.CodeHash)) != 1 {
		if err := s.repo.IncrementResetAttempts(ctx, reset.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, reset.ID, stored.User.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

// DeleteAccount permanently removes the user and all associated data after
// re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	var after func()
	if s.hooks.BeforeDelete != nil {
		after, err = s.hooks.BeforeDelete(ctx, userID)
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
	if after != nil {
		after()
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	sessionID, err := s.repo.CreateSession(ctx, user.ID, hashToken(refreshToken), userAgent, ipAddress, s.now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := s.signAccessToken(user.ID, sessionID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *Service) signAccessToken(userID, sessionID string) (string, error) {
	now := s.now()
	claims := AccessTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    accessTokenIssuer,
			Subject:   accessTokenSubject,
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
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
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func hashResetCode(userID, code string) string {
	sum := sha256.Sum256([]byte(userID + ":" + code))
	return hex.EncodeToString(sum[:])
}
