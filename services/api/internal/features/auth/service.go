package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	perr "example.com/api/internal/platform/errors"
	platformmail "example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetCodeTTL    = 30 * time.Minute

	// MinAge is the legal minimum to use the service; enforced server-side at registration.
	MinAge = 18
	MaxAge = 99

	maxPasswordBytes = 72 // bcrypt limit
	minPasswordRunes = 8
)

var nowFunc = time.Now

// dummyHash lets Login spend the same time for unknown and known e-mails.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("alma-dummy-password"), bcrypt.DefaultCost)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    platformmail.Mailer
	storage   storage.Storage
}

func NewService(repo Repository, jwtSecret []byte, mailer platformmail.Mailer, store storage.Storage) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: mailer, storage: store}
}

// SessionActive tells the middleware whether an access token's session is still valid.
func (s *Service) SessionActive(ctx context.Context, sessionID string) (bool, error) {
	return s.repo.SessionActive(ctx, sessionID)
}

func errInvalidCredentials() error {
	return perr.Unauthorized("invalid_credentials", "invalid email or password")
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, userAgent, ip string) (Tokens, User, error) {
	email, err := NormalizeEmail(req.Email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := validatePassword(req.Password); err != nil {
		return Tokens{}, User{}, err
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return Tokens{}, User{}, perr.Invalid("invalid_birth_date", "birth date must use the YYYY-MM-DD format")
	}
	age := AgeOn(birth, nowFunc())
	if age < MinAge {
		return Tokens{}, User{}, perr.Forbidden("underage", "you must be at least 18 years old")
	}
	if age > MaxAge {
		return Tokens{}, User{}, perr.Invalid("invalid_birth_date", "birth date is not valid")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	user, err := s.repo.CreateUser(ctx, email, string(hash), birth)
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, perr.Conflict("email_taken", "an account already exists for this email")
		}
		return Tokens{}, User{}, err
	}
	tokens, err := s.issueSessionTokens(ctx, user.ID, userAgent, ip)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, errInvalidCredentials()
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return Tokens{}, User{}, errInvalidCredentials()
		}
		return Tokens{}, User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return Tokens{}, User{}, errInvalidCredentials()
	}
	if stored.Status != "active" {
		return Tokens{}, User{}, perr.Forbidden("account_suspended", "this account has been suspended")
	}
	tokens, err := s.issueSessionTokens(ctx, stored.User.ID, userAgent, ip)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, stored.User, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	invalid := perr.Unauthorized("invalid_refresh_token", "invalid refresh token")
	hash := hashToken(strings.TrimSpace(refreshToken))
	if hash == "" {
		return Tokens{}, User{}, invalid
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, hash)
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
	next, err := generateToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(next), userAgent, ip, nowFunc().Add(refreshTokenTTL), nowFunc())
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
	hash := hashToken(strings.TrimSpace(refreshToken))
	if hash == "" {
		return perr.Invalid("invalid_refresh_token", "invalid refresh token")
	}
	err := s.repo.RevokeSessionByTokenHash(ctx, hash, nowFunc())
	if err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, perr.Unauthorized("user_not_found", "account not found")
		}
		return User{}, err
	}
	return user, nil
}

// ForgotPassword always succeeds from the caller's perspective to avoid account enumeration.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized, err := NormalizeEmail(email)
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
	raw := make([]byte, 10)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw) // 16 chars, 80 bits
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashToken(code), nowFunc().Add(resetCodeTTL)); err != nil {
		return err
	}
	display := code[0:4] + "-" + code[4:8] + "-" + code[8:12] + "-" + code[12:16]
	body := fmt.Sprintf("Voici votre code de réinitialisation Alma : %s\n\nIl est valable 30 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.", display)
	go func() {
		if err := s.mailer.Send(normalized, "Votre code de réinitialisation Alma", body); err != nil {
			log.Printf(`{"event":"mail_error","operation":"password_reset","error":%q}`, err.Error())
		}
	}()
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, code, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	normalized := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(code))
	if len(normalized) != 16 {
		return perr.Invalid("invalid_code", "this reset code is invalid or expired")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumePasswordReset(ctx, hashToken(normalized), string(hash), nowFunc()); err != nil {
		if errors.Is(err, ErrRepositoryResetInvalid) {
			return perr.Invalid("invalid_code", "this reset code is invalid or expired")
		}
		return err
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return mapUserNotFound(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return perr.Forbidden("wrong_password", "current password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordRevokeOthers(ctx, userID, string(hash), sessionID, nowFunc())
}

// DeleteAccount erases the account and every associated row (cascade) and its photo files.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return mapUserNotFound(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return perr.Forbidden("wrong_password", "password is incorrect")
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := s.storage.Delete(key); err != nil {
			log.Printf(`{"event":"storage_error","operation":"delete_account_media","error":%q}`, err.Error())
		}
	}
	return nil
}

func mapUserNotFound(err error) error {
	if errors.Is(err, ErrRepositoryNotFound) {
		return perr.Unauthorized("user_not_found", "account not found")
	}
	return err
}

func (s *Service) issueSessionTokens(ctx context.Context, userID, userAgent, ip string) (Tokens, error) {
	refresh, err := generateToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, userID, hashToken(refresh), userAgent, ip, nowFunc().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := signAccessToken(s.jwtSecret, userID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func signAccessToken(secret []byte, userID, sessionID string) (string, error) {
	now := nowFunc()
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

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	if strings.TrimSpace(token) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NormalizeEmail trims, lowercases and validates an e-mail address.
func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" || len(normalized) > 254 {
		return "", perr.Invalid("invalid_email", "email address is not valid")
	}
	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized || !strings.Contains(normalized[strings.LastIndex(normalized, "@"):], ".") {
		return "", perr.Invalid("invalid_email", "email address is not valid")
	}
	return normalized, nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordRunes {
		return perr.Invalid("weak_password", "password must be at least 8 characters")
	}
	if len(password) > maxPasswordBytes {
		return perr.Invalid("password_too_long", "password must be at most 72 bytes")
	}
	return nil
}

// AgeOn returns the full years between birth and now.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}
