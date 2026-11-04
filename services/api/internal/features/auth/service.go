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
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidResetCode    = errors.New("invalid or expired code")
)

// ValidationError is returned for unacceptable user input.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetCodeTTL    = 15 * time.Minute
	maxResetTries   = 5
)

// dummyHash equalises login timing when the email is unknown.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mail      mailer.Mailer
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, mail: m}
}

// IsSessionActive implements middleware.SessionChecker.
func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// NormalizeEmail trims and lowercases an address and validates its syntax.
func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized || len(normalized) > 254 || !strings.Contains(normalized[strings.LastIndex(normalized, "@"):], ".") {
		return "", &ValidationError{"invalid email address"}
	}
	return normalized, nil
}

// ValidatePassword enforces a minimal strength policy.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return &ValidationError{"password must be between 8 and 72 characters"}
	}
	var letter, digit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !letter || !digit {
		return &ValidationError{"password must contain at least one letter and one digit"}
	}
	return nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalized, err := NormalizeEmail(email)
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
	user, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
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
	return tokens, stored.User, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, hashToken(strings.TrimSpace(refreshToken)))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	user, err := s.repo.GetUserByID(ctx, session.UserID)
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
	if err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(next), userAgent, ipAddress, time.Now().Add(refreshTokenTTL)); err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	access, err := signAccessToken(s.jwtSecret, user.ID, session.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	err := s.repo.RevokeSessionByTokenHash(ctx, hashToken(strings.TrimSpace(refreshToken)))
	if err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return User{}, ErrUserNotFound
	}
	return user, err
}

// ForgotPassword emails a one-time code. It never reveals whether the address exists.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserAuthByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	code, err := numericCode(8)
	if err != nil {
		return err
	}
	if err := s.repo.CreateReset(ctx, stored.User.ID, hashToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Votre code de réinitialisation : %s\n\nIl expire dans 15 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.", code)
	return s.mail.Send(ctx, stored.User.Email, "Réinitialisation de votre mot de passe", body)
}

// ResetPassword validates the code, replaces the password and revokes all sessions.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	reset, err := s.repo.GetActiveReset(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if reset.Attempts >= maxResetTries {
		return ErrInvalidResetCode
	}
	if err := s.repo.RegisterResetAttempt(ctx, reset.ID); err != nil {
		return err
	}
	given := hashToken(strings.TrimSpace(code))
	if subtle.ConstantTimeCompare([]byte(given), []byte(reset.CodeHash)) != 1 {
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.CompleteReset(ctx, reset.ID, stored.User.ID, string(hash))
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refresh, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashToken(refresh), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := signAccessToken(s.jwtSecret, user.ID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func signAccessToken(secret []byte, userID, sessionID string) (string, error) {
	now := time.Now()
	claims := AccessTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func numericCode(digits int) (string, error) {
	var sb strings.Builder
	for i := 0; i < digits; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		sb.WriteByte(byte('0' + n.Int64()))
	}
	return sb.String(), nil
}

func hashToken(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
