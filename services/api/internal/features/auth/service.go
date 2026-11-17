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

	"example.com/api/internal/platform/authtoken"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrWeakPassword        = errors.New("weak password")
	ErrInvalidResetCode    = errors.New("invalid reset code")
)

const (
	AccessTokenTTL       = 15 * time.Minute
	refreshTokenTTL      = 30 * 24 * time.Hour
	resetCodeTTL         = 30 * time.Minute
	maxResetCodeAttempts = 5
	resetCodeAlphabet    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	resetCodeLength      = 8
	minPasswordLength    = 8
	maxPasswordBytes     = 72 // bcrypt limit
)

// Disconnector closes realtime connections of a deleted account.
type Disconnector interface {
	Disconnect(userID string)
}

type Service struct {
	repo         Repository
	jwtSecret    []byte
	mailer       mailer.Mailer
	store        storage.Store
	disconnector Disconnector
	dummyHash    []byte
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, store storage.Store, d Disconnector) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equalisation-password"), bcrypt.DefaultCost)
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: m, store: store, disconnector: d, dummyHash: dummy}
}

// ValidatePassword enforces the password policy shared by register, reset and change.
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

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

var ErrInvalidEmail = errors.New("invalid email")

// ValidateEmail checks a normalised address: a single plain "local@domain.tld" with no display name.
func ValidateEmail(email string) error {
	if len(email) < 3 || len(email) > 254 {
		return ErrInvalidEmail
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || parsed.Name != "" {
		return ErrInvalidEmail
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || !strings.Contains(email[at+1:], ".") {
		return ErrInvalidEmail
	}
	return nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return Tokens{}, User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}

	created, err := s.repo.CreateUser(ctx, NormalizeEmail(email), string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}

	user := toUser(created)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			// Equalise timing so account existence cannot be probed.
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}

	user := toUser(stored)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	tokenHash := hashSecret(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, tokenHash)
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

	nextRefresh, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashSecret(nextRefresh), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	user := toUser(stored)
	access, err := authtoken.Sign(s.jwtSecret, user.ID, rotated.ID, AccessTokenTTL, time.Now())
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: nextRefresh}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashSecret(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return ErrInvalidRefreshToken
	}
	if err := s.repo.RevokeSessionByTokenHash(ctx, tokenHash); err != nil && !errors.Is(err, ErrRepositorySessionGone) {
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

// RequestPasswordReset always succeeds from the caller's point of view so that the
// endpoint cannot be used to enumerate accounts.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
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
	if err := s.repo.CreateResetCode(ctx, stored.ID, hashSecret(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	msg := mailer.Message{
		To:      stored.Email,
		Subject: "Ton code de réinitialisation Lueur",
		Body: fmt.Sprintf("Bonjour,\n\nTon code de réinitialisation est : %s\nIl expire dans 30 minutes.\n\n"+
			"Si tu n'es pas à l'origine de cette demande, ignore simplement cet e-mail.\n", code),
	}
	if err := s.mailer.Send(ctx, msg); err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","user_id":%q}`, stored.ID)
		return err
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
	active, err := s.repo.GetActiveResetCode(ctx, stored.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if active.Attempts >= maxResetCodeAttempts {
		return ErrInvalidResetCode
	}
	normalizedCode := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	if subtle.ConstantTimeCompare([]byte(hashSecret(normalizedCode)), []byte(active.CodeHash)) != 1 {
		if err := s.repo.IncrementResetAttempts(ctx, active.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}
	if err := s.repo.ConsumeResetCode(ctx, active.ID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, stored.ID, string(hash)); err != nil {
		return err
	}
	// A reset signs the account out everywhere.
	return s.repo.RevokeAllSessions(ctx, stored.ID, "")
}

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
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	return s.repo.RevokeAllSessions(ctx, userID, sessionID)
}

// DeleteAccount permanently erases the account and its photo files after re-checking the password.
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
		return err
	}
	for _, key := range keys {
		if err := s.store.Delete(ctx, key); err != nil {
			log.Printf(`{"event":"account_delete_photo_cleanup_failed","storage_key":%q}`, key)
		}
	}
	if s.disconnector != nil {
		s.disconnector.Disconnect(userID)
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refresh, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashSecret(refresh), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := authtoken.Sign(s.jwtSecret, user.ID, session.ID, AccessTokenTTL, time.Now())
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomCode() (string, error) {
	var sb strings.Builder
	max := big.NewInt(int64(len(resetCodeAlphabet)))
	for i := 0; i < resetCodeLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		sb.WriteByte(resetCodeAlphabet[n.Int64()])
	}
	return sb.String(), nil
}

func hashSecret(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, IsDemo: stored.IsDemo, CreatedAt: stored.CreatedAt}
}
