package auth

import (
	"context"
	"crypto/hmac"
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

	"example.com/api/internal/platform/mailer"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrInvalidEmail        = errors.New("invalid e-mail address")
)

const (
	accessTokenTTL       = 15 * time.Minute
	refreshTokenTTL      = 30 * 24 * time.Hour
	passwordResetTTL     = 15 * time.Minute
	maxResetCodeAttempts = 5
)

// dummyPasswordHash is compared against when an e-mail is unknown so login
// timing does not reveal which addresses have an account.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-password"), bcrypt.DefaultCost)

// AccountCleaner removes data stored outside Postgres (photo files) when an
// account is deleted.
type AccountCleaner interface {
	DeleteUserMedia(ctx context.Context, userID string) error
}

// Disconnector closes realtime connections of a user.
type Disconnector interface {
	Disconnect(userID string)
}

type Service struct {
	repo         Repository
	jwtSecret    []byte
	mailer       mailer.Mailer
	cleaner      AccountCleaner
	disconnector Disconnector
}

type Options struct {
	Mailer       mailer.Mailer
	Cleaner      AccountCleaner
	Disconnector Disconnector
}

func NewService(repo Repository, jwtSecret []byte, opts Options) *Service {
	if opts.Mailer == nil {
		opts.Mailer = mailer.LogMailer{}
	}
	return &Service{
		repo:         repo,
		jwtSecret:    jwtSecret,
		mailer:       opts.Mailer,
		cleaner:      opts.Cleaner,
		disconnector: opts.Disconnector,
	}
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
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

// IsSessionActive implements middleware.SessionValidator.
func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// DeleteAccount permanently removes the user after re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	hash, err := s.repo.GetPasswordHashByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if s.cleaner != nil {
		if err := s.cleaner.DeleteUserMedia(ctx, userID); err != nil {
			// The account is already gone; orphaned files are unreachable
			// (no DB row, no signed URL) and are reported for cleanup.
			return fmt.Errorf("account deleted but media cleanup failed: %w", err)
		}
	}
	if s.disconnector != nil {
		s.disconnector.Disconnect(userID)
	}
	return nil
}

// RequestPasswordReset e-mails a 6-digit code. Unknown addresses are ignored
// silently so the endpoint cannot be used to enumerate accounts.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
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
	if _, err := s.repo.CreatePasswordReset(ctx, stored.User.ID, s.hashResetCode(stored.User.ID, code), time.Now().Add(passwordResetTTL)); err != nil {
		return err
	}
	body := "Votre code de réinitialisation est : " + code + "\n\n" +
		"Il expire dans 15 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message."
	return s.mailer.Send(ctx, stored.User.Email, "Réinitialisation de votre mot de passe", body)
}

// ResetPassword verifies the code (max 5 attempts) and sets a new password.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
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
	if reset.Attempts >= maxResetCodeAttempts {
		return ErrInvalidResetCode
	}
	expected := s.hashResetCode(stored.User.ID, strings.TrimSpace(code))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(reset.CodeHash)) != 1 {
		if err := s.repo.IncrementPasswordResetAttempts(ctx, reset.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, reset.ID, stored.User.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if s.disconnector != nil {
		s.disconnector.Disconnect(stored.User.ID)
	}
	return nil
}

// hashResetCode keys the code with the server secret so a database leak does
// not allow brute-forcing the small 6-digit space offline.
func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte("password-reset-v1|" + userID + "|" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateResetCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

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

// NormalizeEmail trims, lower-cases and validates a bare e-mail address.
func NormalizeEmail(email string) (string, error) {
	normalized := normalizeEmail(email)
	if len(normalized) > 254 {
		return "", ErrInvalidEmail
	}
	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Address != normalized || !strings.Contains(normalized[strings.LastIndex(normalized, "@")+1:], ".") {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

func toUser(stored StoredUser) User {
	return User{
		ID:             stored.ID,
		Email:          stored.Email,
		OrganizationID: stored.OrganizationID,
		CreatedAt:      stored.CreatedAt,
	}
}
