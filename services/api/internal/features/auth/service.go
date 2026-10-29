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
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 30 * 24 * time.Hour
	resetCodeTTL     = 30 * time.Minute
	resetCooldown    = 60 * time.Second
	resetMaxAttempts = 5
	resetCodeLength  = 8
	resetAlphabet    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	minPasswordLen   = 8
	maxPasswordBytes = 72 // bcrypt silently truncates beyond this
)

// dummyHash lets Login spend the same time on unknown emails as on wrong passwords.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer

	// RemoveFiles deletes stored photo files after an account is deleted.
	RemoveFiles func(keys []string)
	// OnAccountDeleted lets other components (realtime hub) drop per-user state.
	OnAccountDeleted func(userID string)
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: m}
}

var errInvalidCredentials = platformerrors.Unauthorized("invalid email or password")

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := validatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}

	user, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, platformerrors.Conflict("an account with this email already exists")
		}
		return Tokens{}, User{}, err
	}
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, errInvalidCredentials
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return Tokens{}, User{}, errInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, errInvalidCredentials
	}
	tokens, err := s.issueSessionTokens(ctx, stored.User, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, stored.User, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	invalid := platformerrors.Unauthorized("invalid refresh token")
	tokenHash := hashToken(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return Tokens{}, User{}, invalid
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}
	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}

	next, err := randomToken(32)
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(next), userAgent, ipAddress, time.Now().Add(refreshTokenTTL), time.Now())
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}
	access, err := signAccessToken(s.jwtSecret, user.ID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return platformerrors.Invalid("invalid refresh token")
	}
	err := s.repo.RevokeSessionByTokenHash(ctx, tokenHash, time.Now())
	if err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, platformerrors.Unauthorized("user not found")
		}
		return User{}, err
	}
	return user, nil
}

// SessionActive implements middleware.SessionChecker.
func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.SessionActive(ctx, sessionID, userID)
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.verifyPassword(ctx, userID, currentPassword)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, stored.User.ID, string(hash)); err != nil {
		return err
	}
	return nil
}

// ForgotPassword emails a short one-time code. It always succeeds from the
// caller's point of view so the endpoint cannot be used to discover accounts.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	if latest, err := s.repo.GetLatestPasswordReset(ctx, stored.User.ID); err == nil && time.Since(latest.CreatedAt) < resetCooldown {
		return nil
	}

	code, err := randomCode(resetCodeLength)
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}

	body := "Votre code de réinitialisation : " + code + "\n\n" +
		"Il est valable 30 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez cet email : votre mot de passe reste inchangé."
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.mailer.Send(sendCtx, normalized, "Réinitialisation de votre mot de passe", body); err != nil {
			log.Printf(`{"event":"mail_error","operation":"password_reset","error":%q}`, err.Error())
		}
	}()
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	invalid := platformerrors.Invalid("invalid or expired code")
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	normalized, err := normalizeEmail(email)
	if err != nil {
		return invalid
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return invalid
		}
		return err
	}
	reset, err := s.repo.GetLatestPasswordReset(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return invalid
		}
		return err
	}
	if reset.Attempts >= resetMaxAttempts {
		return invalid
	}

	candidate := hashToken(normalizeCode(code))
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(reset.CodeHash)) != 1 {
		if err := s.repo.RegisterResetAttempt(ctx, reset.ID); err != nil {
			return err
		}
		return invalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumePasswordReset(ctx, reset.ID, stored.User.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return invalid
		}
		return err
	}
	return nil
}

// DeleteAccount permanently removes the account and everything attached to it.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	if _, err := s.verifyPassword(ctx, userID, password); err != nil {
		return err
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	if s.RemoveFiles != nil {
		s.RemoveFiles(keys)
	}
	if s.OnAccountDeleted != nil {
		s.OnAccountDeleted(userID)
	}
	return nil
}

func (s *Service) verifyPassword(ctx context.Context, userID, password string) (StoredUserWithPassword, error) {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return StoredUserWithPassword{}, platformerrors.Unauthorized("user not found")
		}
		return StoredUserWithPassword{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return StoredUserWithPassword{}, platformerrors.Forbidden("incorrect password")
	}
	return stored, nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refresh, err := randomToken(32)
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

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomCode(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i, b := range buf {
		out[i] = resetAlphabet[int(b)%len(resetAlphabet)] // 256 % 32 == 0: no modulo bias
	}
	return string(out), nil
}

func normalizeCode(code string) string {
	code = strings.ToUpper(code)
	return strings.NewReplacer(" ", "", "-", "").Replace(code)
}

func hashToken(token string) string {
	if strings.TrimSpace(token) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed || len(trimmed) > 254 || !strings.Contains(trimmed[strings.LastIndex(trimmed, "@"):], ".") {
		return "", platformerrors.Invalid("invalid email address")
	}
	return trimmed, nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLen {
		return platformerrors.Invalid("password must be at least %d characters", minPasswordLen)
	}
	if len(password) > maxPasswordBytes {
		return platformerrors.Invalid("password must be at most %d bytes", maxPasswordBytes)
	}
	if strings.TrimSpace(password) == "" {
		return platformerrors.Invalid("password cannot be blank")
	}
	return nil
}
