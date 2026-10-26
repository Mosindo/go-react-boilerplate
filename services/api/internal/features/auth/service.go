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
	ErrInvalidEmail        = errors.New("please enter a valid email address")
	ErrWeakPassword        = errors.New("password must be 8-72 characters and contain a letter and a digit")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrAccountSuspended    = errors.New("account suspended")
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 30 * 24 * time.Hour
	resetCodeTTL     = 30 * time.Minute
	resetMaxAttempts = 5
	resetCooldown    = time.Minute
)

// dummyHash lets Login spend the same bcrypt time for unknown emails, so
// response timing does not reveal which emails are registered.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser-password"), bcrypt.DefaultCost)

type Service struct {
	repo          Repository
	jwtSecret     []byte
	mailer        mailer.Mailer
	onAccountGone func(userID string)
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: m}
}

// OnAccountDeleted registers a hook (e.g. close realtime connections).
func (s *Service) OnAccountDeleted(fn func(userID string)) { s.onAccountGone = fn }

func ValidatePassword(password string) error {
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

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	if !ValidEmail(normalizeEmail(email)) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}

	created, err := s.repo.CreateUser(ctx, normalizeEmail(email), string(hash))
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
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
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
	if stored.User.Status != "active" {
		return Tokens{}, User{}, ErrAccountSuspended
	}

	user := toUser(stored.User)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	currentHash := hashRefreshToken(strings.TrimSpace(refreshToken))
	if currentHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}

	session, err := s.repo.GetActiveSessionByTokenHash(ctx, currentHash)
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
	if userRecord.Status != "active" {
		return Tokens{}, User{}, ErrAccountSuspended
	}

	refreshSecret, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	now := time.Now()
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, currentHash, hashRefreshToken(refreshSecret),
		userAgent, ipAddress, now.Add(refreshTokenTTL), now)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	user := toUser(userRecord)
	accessToken, err := signAccessToken(s.jwtSecret, user, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshSecret}, user, nil
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

// ForgotPassword emails a one-time 8-digit code. It always succeeds from the
// caller's point of view so it cannot be used to discover registered emails.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	// One code per minute and account: stops the endpoint from being used to spam a mailbox.
	if last, err := s.repo.GetLatestPasswordReset(ctx, stored.User.ID); err == nil && time.Since(last.CreatedAt) < resetCooldown {
		return nil
	}
	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashResetCode(stored.User.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Votre code de réinitialisation : %s\n\nIl est valable %d minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez cet email.", code, int(resetCodeTTL.Minutes()))
	return s.mailer.Send(ctx, stored.User.Email, "Réinitialisation de votre mot de passe", body)
}

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
	reset, err := s.repo.GetLatestPasswordReset(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryResetGone) {
			return ErrInvalidResetCode
		}
		return err
	}
	if reset.Attempts >= resetMaxAttempts {
		return ErrInvalidResetCode
	}
	expected := hashResetCode(stored.User.ID, strings.TrimSpace(code))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(reset.CodeHash)) != 1 {
		_ = s.repo.BumpPasswordResetAttempts(ctx, reset.ID)
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, stored.User.ID, reset.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryResetGone) {
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
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

// DeleteAccount permanently removes the user; every related row (profile,
// photos, swipes, matches, messages, notifications, sessions) cascades.
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
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if s.onAccountGone != nil {
		s.onAccountGone(userID)
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashRefreshToken(refreshToken), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := signAccessToken(s.jwtSecret, user, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func signAccessToken(secret []byte, user User, sessionID string) (string, error) {
	now := time.Now()
	claims := AccessTokenClaims{
		UserID:    user.ID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateResetCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%08d", n.Int64()), nil
}

func hashResetCode(userID, code string) string {
	sum := sha256.Sum256([]byte(userID + ":" + code))
	return hex.EncodeToString(sum[:])
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

// ValidEmail reports whether the (already trimmed) address is a plain
// addr-spec: no display name, no spaces, one @ and a dotted domain.
func ValidEmail(email string) bool {
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
