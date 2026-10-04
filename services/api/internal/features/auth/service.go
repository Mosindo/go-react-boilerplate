package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	platformmail "example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 30 * 24 * time.Hour
	resetTokenTTL    = time.Hour
	minPasswordLen   = 8
	maxPasswordBytes = 72 // bcrypt silently/explicitly stops at 72 bytes
)

type Service struct {
	repo       Repository
	jwtSecret  []byte
	mailer     platformmail.Mailer
	files      storage.Store
	appBaseURL string
	// dummyHash lets Login spend the same bcrypt time for unknown emails (no user enumeration by timing).
	dummyHash []byte
}

type Deps struct {
	Mailer     platformmail.Mailer
	Files      storage.Store
	AppBaseURL string
}

func NewService(repo Repository, jwtSecret []byte, deps Deps) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	return &Service{
		repo:       repo,
		jwtSecret:  jwtSecret,
		mailer:     deps.Mailer,
		files:      deps.Files,
		appBaseURL: deps.AppBaseURL,
		dummyHash:  dummy,
	}
}

var errInvalidCredentials = platformerrors.Unauthorized("Invalid email or password")

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := validatePassword(password, "password"); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	created, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, platformerrors.Conflict("conflict", "An account with this email already exists")
		}
		return Tokens{}, User{}, err
	}
	user := toUser(created)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Tokens{}, User{}, errInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, errInvalidCredentials
	}
	if stored.User.Status != "active" {
		return Tokens{}, User{}, platformerrors.Forbidden("This account is suspended")
	}
	user := toUser(stored.User)
	_ = s.repo.TouchUser(ctx, user.ID)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	invalid := platformerrors.Unauthorized("Invalid refresh token")
	current := hashToken(strings.TrimSpace(refreshToken))
	if current == "" {
		return Tokens{}, User{}, invalid
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, current)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}
	next, err := generateToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, current, hashToken(next), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByID(ctx, rotated.UserID)
	if err != nil {
		return Tokens{}, User{}, invalid
	}
	_ = s.repo.TouchUser(ctx, stored.ID)
	access, err := signAccessToken(s.jwtSecret, stored.ID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, toUser(stored), nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	hash := hashToken(strings.TrimSpace(refreshToken))
	if hash == "" {
		return platformerrors.FieldInvalid("refreshToken", "required")
	}
	if err := s.repo.RevokeSessionByTokenHash(ctx, hash); err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, platformerrors.Unauthorized("Account not found")
		}
		return User{}, err
	}
	return toUser(stored), nil
}

// ValidateSession backs middleware.RequireUser.
func (s *Service) ValidateSession(ctx context.Context, userID, sessionID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, userID, sessionID)
}

// ForgotPassword always succeeds from the caller's point of view (no account enumeration).
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
	token, err := generateToken()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashToken(token), time.Now().Add(resetTokenTTL)); err != nil {
		return err
	}
	body := "Someone asked to reset the password of your account.\n\n" +
		"Reset code (valid for 1 hour): " + token + "\n"
	if s.appBaseURL != "" {
		body += fmt.Sprintf("\nOr open: %s/reset-password?token=%s\n", s.appBaseURL, token)
	}
	body += "\nIf this was not you, ignore this email: your password stays unchanged."
	if err := s.mailer.Send(ctx, stored.User.Email, "Reset your password", body); err != nil {
		// Delivery failures must not leak whether the account exists: log and answer as usual.
		log.Printf(`{"event":"reset_mail_failed","error":%q}`, err.Error())
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if err := validatePassword(password, "password"); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := s.repo.ConsumePasswordReset(ctx, hashToken(strings.TrimSpace(token)), string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryResetGone) {
			return platformerrors.FieldInvalid("token", "This reset code is invalid or has expired")
		}
		return err
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := validatePassword(next, "newPassword"); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return platformerrors.Unauthorized("Account not found")
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return platformerrors.FieldInvalid("currentPassword", "Current password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, userID, string(hash)); err != nil {
		return err
	}
	return s.repo.RevokeUserSessions(ctx, userID, sessionID)
}

// DeleteAccount permanently removes the user and every piece of data tied to them.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return platformerrors.Unauthorized("Account not found")
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return platformerrors.FieldInvalid("password", "Password is incorrect")
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := s.files.Delete(ctx, key); err != nil {
			log.Printf(`{"event":"photo_cleanup_failed","key":%q,"error":%q}`, key, err.Error())
		}
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ip string) (Tokens, error) {
	refresh, err := generateToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashToken(refresh), userAgent, ip, time.Now().Add(refreshTokenTTL))
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
	claims := jwt.MapClaims{
		"uid": userID,
		"sid": sessionID,
		"iat": now.Unix(),
		"exp": now.Add(accessTokenTTL).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > 254 {
		return "", platformerrors.FieldInvalid("email", "Enter a valid email address")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" {
		return "", platformerrors.FieldInvalid("email", "Enter a valid email address")
	}
	at := strings.LastIndex(email, "@")
	if !strings.Contains(email[at+1:], ".") {
		return "", platformerrors.FieldInvalid("email", "Enter a valid email address")
	}
	return email, nil
}

func validatePassword(password, field string) error {
	switch {
	case len([]rune(password)) < minPasswordLen:
		return platformerrors.FieldInvalid(field, "Use at least 8 characters")
	case len(password) > maxPasswordBytes:
		return platformerrors.FieldInvalid(field, "Use at most 72 bytes")
	}
	return nil
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, ProfileCompleted: stored.ProfileCompleted, CreatedAt: stored.CreatedAt}
}
