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
	"strings"
	"time"

	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrWeakPassword        = errors.New("password must be 8 to 72 bytes long")
	ErrInvalidResetCode    = errors.New("invalid or expired code")
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 30 * 24 * time.Hour
	resetCodeTTL     = 30 * time.Minute
	resetCooldown    = time.Minute
	maxResetAttempts = 5
	resetCodeLength  = 10
	// Unambiguous base32-like alphabet (no 0/O/1/I/L).
	resetAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

// AccountHook runs before an account is deleted. The returned function runs after a successful deletion
// (used by other features to notify the people the user was matched with).
type AccountHook func(ctx context.Context, userID string) (after func(), err error)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mail      mailer.Mailer
	hooks     []AccountHook
	cost      int
	// dummyHash is compared against when the e-mail is unknown, so login timing does not reveal accounts.
	dummyHash []byte
}

// NewService builds the auth service. bcryptCost <= 0 selects bcrypt.DefaultCost (tests pass bcrypt.MinCost).
func NewService(repo Repository, jwtSecret []byte, mail mailer.Mailer, bcryptCost int, hooks ...AccountHook) *Service {
	if bcryptCost <= 0 {
		bcryptCost = bcrypt.DefaultCost
	}
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
	return &Service{repo: repo, jwtSecret: jwtSecret, mail: mail, hooks: hooks, cost: bcryptCost, dummyHash: dummy}
}

func validatePassword(password string) error {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordBytes {
		return ErrWeakPassword
	}
	return nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	if err := validatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
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
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
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
	_ = s.repo.TouchUser(ctx, user.ID)
	return tokens, user, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, hashRefreshToken(strings.TrimSpace(refreshToken)))
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
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashRefreshToken(refreshSecret), userAgent, ipAddress,
		time.Now().Add(refreshTokenTTL), time.Now())
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
	_ = s.repo.TouchUser(ctx, user.ID)
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

// IsSessionActive implements middleware.SessionChecker.
func (s *Service) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, userID, sessionID)
}

// ForgotPassword never reveals whether the e-mail exists: errors are logged, not returned.
func (s *Service) ForgotPassword(ctx context.Context, email string) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if !errors.Is(err, ErrRepositoryNotFound) {
			log.Printf(`{"event":"password_forgot_error","error":%q}`, err.Error())
		}
		return
	}
	code, err := generateResetCode()
	if err != nil {
		log.Printf(`{"event":"password_forgot_error","error":%q}`, err.Error())
		return
	}
	created, err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashResetCode(code), time.Now().Add(resetCodeTTL), resetCooldown)
	if err != nil {
		log.Printf(`{"event":"password_forgot_error","error":%q}`, err.Error())
		return
	}
	if !created {
		return
	}
	body := "Votre code de réinitialisation : " + code + "\n\nIl est valable 30 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message."
	if err := s.mail.Send(ctx, stored.User.Email, "Votre code de réinitialisation", body); err != nil {
		log.Printf(`{"event":"password_forgot_mail_error","error":%q}`, err.Error())
	}
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	reset, err := s.repo.LatestPasswordReset(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if reset.Attempts >= maxResetAttempts {
		return ErrInvalidResetCode
	}
	if err := s.repo.IncrementResetAttempts(ctx, reset.ID); err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(hashResetCode(normalizeResetCode(code))), []byte(reset.CodeHash)) != 1 {
		return ErrInvalidResetCode
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.cost)
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

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), s.cost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

// DeleteAccount removes the user and, through ON DELETE CASCADE, every row that belongs to them.
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

	afters := make([]func(), 0, len(s.hooks))
	for _, hook := range s.hooks {
		after, err := hook(ctx, userID)
		if err != nil {
			return err
		}
		if after != nil {
			afters = append(afters, after)
		}
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}
	for _, after := range afters {
		after()
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
	claims := jwt.MapClaims{
		"uid": user.ID,
		"sid": sessionID,
		"exp": jwt.NewNumericDate(now.Add(accessTokenTTL)),
		"iat": jwt.NewNumericDate(now),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
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
		// 31 symbols: modulo bias is negligible for a 5-attempt, 30-minute code.
		out[i] = resetAlphabet[int(b)%len(resetAlphabet)]
	}
	return string(out), nil
}

func normalizeResetCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	return strings.ReplaceAll(code, " ", "")
}

func hashResetCode(code string) string {
	sum := sha256.Sum256([]byte(code))
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

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, ProfileComplete: stored.ProfileComplete, CreatedAt: stored.CreatedAt}
}
