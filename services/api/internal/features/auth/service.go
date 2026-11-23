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
	ErrWeakPassword        = errors.New("password must be 8-128 characters and contain a letter and a digit")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrWrongPassword       = errors.New("wrong password")
	ErrInvalidEmail        = errors.New("invalid email address")
)

// StorageCleaner removes binary objects (photos) owned by a deleted account.
type StorageCleaner interface {
	StorageKeysForUser(ctx context.Context, userID string) ([]string, error)
	PurgeObjects(ctx context.Context, keys []string)
}

// Disconnector closes realtime connections of a user.
type Disconnector interface {
	Disconnect(userID string)
}

const (
	accessTokenTTL     = 15 * time.Minute
	refreshTokenTTL    = 30 * 24 * time.Hour
	resetCodeTTL       = 30 * time.Minute
	resetCodeLength    = 8
	resetMaxAttempts   = 5
	resetMaxPerHour    = 3
	resetCodeAlphabet  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	minPasswordLength  = 8
	maxPasswordLength  = 128
	passwordBcryptCost = 12
	resetEmailSubject  = "Ton code de réinitialisation Hearth"
	resetEmailBodyFmt  = "Bonjour,\n\nVoici ton code pour choisir un nouveau mot de passe : %s\n\nIl expire dans 30 minutes. Si tu n'es pas à l'origine de cette demande, ignore simplement ce message.\n\nL'équipe Hearth"
)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	storage   StorageCleaner
	realtime  Disconnector
}

func NewService(repo Repository, jwtSecret []byte) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		mailer:    mailer.DevLog{},
	}
}

// WithMailer sets the transactional mail transport.
func (s *Service) WithMailer(m mailer.Mailer) *Service {
	s.mailer = m
	return s
}

// WithAccountCleanup wires the collaborators used on account deletion.
func (s *Service) WithAccountCleanup(storage StorageCleaner, realtime Disconnector) *Service {
	s.storage = storage
	s.realtime = realtime
	return s
}

// ValidatePassword enforces the password policy.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength || len(password) > maxPasswordLength {
		return ErrWeakPassword
	}
	hasLetter, hasDigit := false, false
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

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), passwordBcryptCost)
	return string(hash), err
}

// NormalizeEmail trims, lowercases and validates an email address.
func NormalizeEmail(raw string) (string, error) {
	email := normalizeEmail(raw)
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

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return Tokens{}, User{}, err
	}

	created, err := s.repo.CreateUser(ctx, normalizedEmail, hash)
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
	normalizedEmail := normalizeEmail(email)

	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			// Equalize timing with the existing-user path to avoid account enumeration.
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}

	user := toUser(stored.User)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}

	return tokens, user, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	tokenHash := hashRefreshToken(strings.TrimSpace(refreshToken))
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	userRecord, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrUserNotFound
		}
		return Tokens{}, User{}, err
	}

	refreshSecret, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	nextExpiry := time.Now().Add(refreshTokenTTL)
	rotatedSession, err := s.repo.RotateSessionToken(
		ctx,
		session.ID,
		hashRefreshToken(refreshSecret),
		userAgent,
		ipAddress,
		nextExpiry,
		time.Now(),
	)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	user := toUser(userRecord)
	accessToken, err := signAccessToken(s.jwtSecret, user, rotatedSession.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshSecret,
	}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashRefreshToken(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return ErrInvalidRefreshToken
	}

	err := s.repo.RevokeSessionByTokenHash(ctx, tokenHash, time.Now())
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

// IsSessionActive lets the auth middleware reject revoked sessions.
func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// RequestPasswordReset emails a one-time code. It never reveals whether the
// email exists.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	recent, err := s.repo.CountRecentPasswordResets(ctx, stored.User.ID, time.Now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= resetMaxPerHour {
		return nil
	}

	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashResetCode(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	return s.mailer.Send(ctx, mailer.Message{
		To:      stored.User.Email,
		Subject: resetEmailSubject,
		Body:    fmt.Sprintf(resetEmailBodyFmt, code),
	})
}

// ResetPassword verifies a reset code, sets the new password and revokes
// every session of the account.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
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
	if reset.Attempts >= resetMaxAttempts {
		return ErrInvalidResetCode
	}
	normalizedCode := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	if subtle.ConstantTimeCompare([]byte(hashResetCode(normalizedCode)), []byte(reset.CodeHash)) != 1 {
		if err := s.repo.IncrementPasswordResetAttempts(ctx, reset.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumePasswordReset(ctx, reset.ID, stored.User.ID, hash); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if s.realtime != nil {
		s.realtime.Disconnect(stored.User.ID)
	}
	return nil
}

// ChangePassword updates the password and revokes every other session.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, currentPassword, newPassword string) error {
	if err := s.verifyPassword(ctx, userID, currentPassword); err != nil {
		return err
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordAndRevokeSessions(ctx, userID, hash, sessionID)
}

// DeleteAccount permanently removes the account and its data after a
// password confirmation.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return err
	}
	var keys []string
	if s.storage != nil {
		var err error
		keys, err = s.storage.StorageKeysForUser(ctx, userID)
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
	if s.storage != nil {
		s.storage.PurgeObjects(context.WithoutCancel(ctx), keys)
	}
	if s.realtime != nil {
		s.realtime.Disconnect(userID)
	}
	return nil
}

func (s *Service) verifyPassword(ctx context.Context, userID, password string) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return ErrWrongPassword
	}
	return nil
}

func generateResetCode() (string, error) {
	buf := make([]byte, resetCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLength)
	for i, b := range buf {
		out[i] = resetCodeAlphabet[int(b)%len(resetCodeAlphabet)]
	}
	return string(out), nil
}

func hashResetCode(code string) string {
	sum := sha256.Sum256([]byte("password-reset:" + code))
	return hex.EncodeToString(sum[:])
}

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-password-1"), passwordBcryptCost)

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}

	session, err := s.repo.CreateSession(
		ctx,
		user.ID,
		hashRefreshToken(refreshToken),
		userAgent,
		ipAddress,
		time.Now().Add(refreshTokenTTL),
	)
	if err != nil {
		return Tokens{}, err
	}

	accessToken, err := signAccessToken(s.jwtSecret, user, session.ID)
	if err != nil {
		return Tokens{}, err
	}

	return Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func signAccessToken(secret []byte, user User, sessionID string) (string, error) {
	now := time.Now()
	claims := AccessTokenClaims{
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		SessionID:      sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashRefreshToken(refreshToken string) string {
	if strings.TrimSpace(refreshToken) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func toUser(stored StoredUser) User {
	return User{
		ID:             stored.ID,
		Email:          stored.Email,
		OrganizationID: stored.OrganizationID,
		CreatedAt:      stored.CreatedAt,
	}
}
