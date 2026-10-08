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

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrAccountSuspended    = errors.New("account suspended")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetCodeTTL    = 15 * time.Minute
	resetMaxTries   = 5
	minPasswordLen  = 8
	maxPasswordLen  = 72 // bcrypt silently/loudly rejects anything longer
	resetAlphabet   = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	resetCodeLen    = 8
	bcryptCost      = 11
)

// FilePurger lets other features remove user-owned files when an account is deleted.
// Prepare runs before the database delete and returns a cleanup invoked after it succeeds.
type FilePurger interface {
	PrepareDelete(ctx context.Context, userID string) (cleanup func(), err error)
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	purger    FilePurger
	// dummyHash equalises login timing for unknown emails.
	dummyHash []byte
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, purger FilePurger) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcryptCost)
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: m, purger: purger, dummyHash: dummy}
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail, err := validateEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := validatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return Tokens{}, User{}, err
	}

	created, err := s.repo.CreateUser(ctx, normalizedEmail, string(hash))
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
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
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
	if userRecord.Status != "active" {
		return Tokens{}, User{}, ErrAccountSuspended
	}

	refreshSecret, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashRefreshToken(refreshSecret),
		userAgent, ipAddress, time.Now().Add(refreshTokenTTL), time.Now())
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

// ForgotPassword always succeeds from the caller's perspective to avoid revealing
// which emails are registered. The 8-character code is emailed, valid 15 minutes.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized := normalizeEmail(email)
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashRefreshToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Votre code de réinitialisation : %s\n\nIl est valable %d minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.", code, int(resetCodeTTL.Minutes()))
	return s.mailer.Send(ctx, stored.User.Email, "Réinitialisation de votre mot de passe", body)
}

var errBadResetCode = platformerrors.BadRequest("invalid_reset_code", "invalid or expired code")

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return errBadResetCode
		}
		return err
	}
	reset, err := s.repo.GetLatestPasswordReset(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return errBadResetCode
		}
		return err
	}
	if reset.Attempts >= resetMaxTries {
		return errBadResetCode
	}
	if err := s.repo.IncrementResetAttempts(ctx, reset.ID); err != nil {
		return err
	}
	given := hashRefreshToken(strings.ToUpper(strings.TrimSpace(code)))
	if subtle.ConstantTimeCompare([]byte(given), []byte(reset.CodeHash)) != 1 {
		return errBadResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumeReset(ctx, reset.ID, stored.User.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return errBadResetCode
		}
		return err
	}
	return nil
}

// ChangePassword verifies the current password then revokes the user's other sessions.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return mapNotFound(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)); err != nil {
		return platformerrors.Forbidden("wrong_password", "current password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcryptCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

// DeleteAccount permanently removes the user; foreign keys cascade to every owned row.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return mapNotFound(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return platformerrors.Forbidden("wrong_password", "password is incorrect")
	}
	cleanup := func() {}
	if s.purger != nil {
		if cleanup, err = s.purger.PrepareDelete(ctx, userID); err != nil {
			return err
		}
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return mapNotFound(err)
	}
	cleanup()
	return nil
}

func mapNotFound(err error) error {
	if errors.Is(err, ErrRepositoryNotFound) {
		return platformerrors.Unauthorized("unauthorized", "user not found")
	}
	return err
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
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func generateResetCode() (string, error) {
	buf := make([]byte, resetCodeLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLen)
	for i, b := range buf {
		out[i] = resetAlphabet[int(b)%len(resetAlphabet)] // 256 % 32 == 0: unbiased
	}
	return string(out), nil
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

func validateEmail(email string) (string, error) {
	normalized := normalizeEmail(email)
	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized || len(normalized) > 254 || !strings.Contains(normalized[strings.LastIndex(normalized, "@"):], ".") {
		return "", platformerrors.BadRequest("invalid_email", "invalid email address")
	}
	return normalized, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return platformerrors.BadRequest("invalid_password", fmt.Sprintf("password must be %d to %d characters", minPasswordLen, maxPasswordLen))
	}
	var hasLetter, hasOther bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else {
			hasOther = true
		}
	}
	if !hasLetter || !hasOther {
		return platformerrors.BadRequest("weak_password", "password must mix letters with digits or symbols")
	}
	return nil
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, CreatedAt: stored.CreatedAt}
}
