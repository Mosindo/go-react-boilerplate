package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"example.com/api/internal/platform/authtoken"
	"example.com/api/internal/platform/mailer"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrWeakPassword        = errors.New("password must contain at least one letter and one digit")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrInvalidEmail        = errors.New("invalid email address")
)

const (
	refreshTokenTTL      = 30 * 24 * time.Hour
	resetCodeTTL         = 30 * time.Minute
	resetCodeCooldown    = time.Minute
	resetCodeMaxAttempts = 5
	resetCodeLength      = 10
	resetCodeAlphabet    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// dummyHash is compared against when an e-mail is unknown so that login
// timing does not reveal whether an account exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-password-1"), bcrypt.DefaultCost)

// AccountPurger releases resources held outside the database (photo files)
// when an account is deleted.
type AccountPurger interface {
	StorageKeysForUser(ctx context.Context, userID string) ([]string, error)
	DeleteStoredObjects(ctx context.Context, keys []string)
}

// Disconnecter closes realtime connections of a user.
type Disconnecter interface {
	DisconnectUser(userID string)
}

type Service struct {
	repo         Repository
	jwtSecret    []byte
	mailer       mailer.Mailer
	purger       AccountPurger
	disconnecter Disconnecter
	now          func() time.Time
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, purger AccountPurger, disconnecter Disconnecter) *Service {
	return &Service{
		repo:         repo,
		jwtSecret:    jwtSecret,
		mailer:       m,
		purger:       purger,
		disconnecter: disconnecter,
		now:          time.Now,
	}
}

func validatePasswordStrength(password string) error {
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit || len(password) < 8 {
		return ErrWeakPassword
	}
	return nil
}

// ValidateEmail normalizes (trim + lowercase) and validates an address.
func ValidateEmail(raw string) (string, error) {
	email := normalizeEmail(raw)
	if len(email) > 254 {
		return "", ErrInvalidEmail
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || parsed.Name != "" {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || !strings.Contains(email[at+1:], ".") || strings.HasSuffix(email, ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func (s *Service) Register(ctx context.Context, rawEmail, password, userAgent, ipAddress string) (Tokens, User, error) {
	email, err := ValidateEmail(rawEmail)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := validatePasswordStrength(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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

	tokens, err := s.issueSessionTokens(ctx, user.ID, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	stored, err := s.repo.GetUserByEmail(ctx, normalizeEmail(email))
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

	tokens, err := s.issueSessionTokens(ctx, stored.ID, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, stored.User, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	currentHash := hashSecret(refreshToken)
	if currentHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	nextSecret, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}

	session, err := s.repo.RotateSession(ctx, currentHash, hashSecret(nextSecret), userAgent, ipAddress, s.now().Add(refreshTokenTTL))
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

	accessToken, err := authtoken.Sign(s.jwtSecret, authtoken.TypeAccess, session.UserID, session.ID, authtoken.AccessTTL, s.now())
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: nextSecret}, stored.User, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashSecret(refreshToken)
	if tokenHash == "" {
		return ErrInvalidRefreshToken
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
	return stored.User, nil
}

// RealtimeTicket issues a 60-second token that only the WebSocket endpoint accepts.
func (s *Service) RealtimeTicket(userID, sessionID string) (string, error) {
	return authtoken.Sign(s.jwtSecret, authtoken.TypeRealtime, userID, sessionID, authtoken.RealtimeTTL, s.now())
}

// RequestPasswordReset always succeeds from the caller's point of view so the
// endpoint cannot be used to discover which e-mails have an account.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalized := normalizeEmail(email)
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	latest, err := s.repo.LatestResetCreatedAt(ctx, stored.ID)
	if err != nil {
		return err
	}
	if latest != nil && s.now().Sub(*latest) < resetCodeCooldown {
		return nil
	}

	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.ID, hashSecret(code), s.now().Add(resetCodeTTL)); err != nil {
		return err
	}

	err = s.mailer.Send(ctx, mailer.Message{
		To:      stored.Email,
		Subject: "Lueur — ton code de réinitialisation",
		Body: "Bonjour,\n\nVoici ton code pour choisir un nouveau mot de passe : " + code +
			"\n\nIl est valable 30 minutes. Si tu n'es pas à l'origine de cette demande, ignore simplement cet e-mail.\n\nL'équipe Lueur",
	})
	if err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","user_id":%q,"error":%q}`, stored.ID, err.Error())
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword, userAgent, ipAddress string) (Tokens, User, error) {
	if err := validatePasswordStrength(newPassword); err != nil {
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidResetCode
		}
		return Tokens{}, User{}, err
	}
	reset, err := s.repo.GetActivePasswordReset(ctx, stored.ID, resetCodeMaxAttempts)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidResetCode
		}
		return Tokens{}, User{}, err
	}
	provided := hashSecret(normalizeResetCode(code))
	if subtle.ConstantTimeCompare([]byte(provided), []byte(reset.CodeHash)) != 1 {
		if err := s.repo.IncrementResetAttempts(ctx, reset.ID); err != nil {
			return Tokens{}, User{}, err
		}
		return Tokens{}, User{}, ErrInvalidResetCode
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := s.repo.CompletePasswordReset(ctx, stored.ID, reset.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidResetCode
		}
		return Tokens{}, User{}, err
	}
	s.disconnecter.DisconnectUser(stored.ID)

	tokens, err := s.issueSessionTokens(ctx, stored.ID, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, stored.User, nil
}

// ChangePassword keeps the current session and signs out every other device.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, currentPassword, newPassword string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

// DeleteAccount permanently removes the account and all associated data.
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

	keys, err := s.purger.StorageKeysForUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	s.purger.DeleteStoredObjects(ctx, keys)
	s.disconnecter.DisconnectUser(userID)
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, userID, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, userID, hashSecret(refreshToken), userAgent, ipAddress, s.now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := authtoken.Sign(s.jwtSecret, authtoken.TypeAccess, userID, session.ID, authtoken.AccessTTL, s.now())
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func generateRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateResetCode() (string, error) {
	buf := make([]byte, resetCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLength)
	for i, b := range buf {
		// 256 is a multiple of 32, so the modulo is unbiased.
		out[i] = resetCodeAlphabet[int(b)%len(resetCodeAlphabet)]
	}
	return string(out), nil
}

func normalizeResetCode(code string) string {
	return strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(code)))
}

func hashSecret(secret string) string {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
