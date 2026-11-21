package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrWeakPassword        = errors.New("password must be 8-128 characters and contain a letter and a digit")
	ErrInvalidResetCode    = errors.New("invalid or expired code")
	ErrWrongPassword       = errors.New("wrong password")
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 30 * 24 * time.Hour
	resetCodeTTL     = 15 * time.Minute
	maxResetAttempts = 5
	maxResetsPerHour = 3
)

// SessionCloser closes live realtime connections when sessions are revoked.
type SessionCloser interface {
	DisconnectUser(userID string)
	DisconnectSession(sessionID string)
}

// dummyHash is compared against when an email is unknown so that login takes
// the same time whether or not the account exists (no user enumeration by timing).
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-password-1"), bcrypt.DefaultCost)

// Options are the optional collaborators of the auth service.
type Options struct {
	Mailer    mailer.Mailer
	Storage   storage.Storage
	Publisher realtime.Publisher
	Sessions  SessionCloser
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	opts      Options
}

func NewService(repo Repository, jwtSecret []byte, opts ...Options) *Service {
	o := Options{}
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Mailer == nil {
		o.Mailer = mailer.Disabled{}
	}
	if o.Publisher == nil {
		o.Publisher = realtime.Nop{}
	}
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		opts:      o,
	}
}

// ValidatePassword enforces the password policy.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
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

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail := normalizeEmail(email)
	if err := ValidatePassword(password); err != nil {
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
		tokenHash,
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

	sessionID, err := s.repo.RevokeSessionByTokenHash(ctx, tokenHash, time.Now())
	if err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	if sessionID != "" && s.opts.Sessions != nil {
		s.opts.Sessions.DisconnectSession(sessionID)
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

// SessionActive is used by the auth middleware to reject tokens whose session
// was revoked (logout, password reset, account deletion).
func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.SessionActive(ctx, sessionID, userID)
}

func (s *Service) resetCodeHash(userID, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte("password-reset:" + userID + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateResetCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", binary.BigEndian.Uint64(buf)%1_000_000), nil
}

// RequestPasswordReset emails a 6-digit code. It never reveals whether the
// email is registered: unknown emails are silently ignored.
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
	userID := stored.User.ID
	// Per-account throttle (independent of the caller's IP).
	recent, err := s.repo.CountRecentResetCodes(ctx, userID, time.Now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= maxResetsPerHour {
		return nil
	}
	if err := s.repo.CreateResetCode(ctx, userID, s.resetCodeHash(userID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	msg := mailer.Message{
		To:      stored.User.Email,
		Subject: "Votre code de réinitialisation",
		Body: "Bonjour,\n\nVotre code de réinitialisation est : " + code +
			"\n\nIl expire dans 15 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.",
	}
	if err := s.opts.Mailer.Send(ctx, msg); err != nil {
		// Logged without the code; the API answer stays identical.
		log.Printf(`{"event":"password_reset_mail_failed","error":%q}`, err.Error())
	}
	return nil
}

// ResetPassword verifies the code (max 5 attempts) and sets a new password.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return ErrInvalidResetCode
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	userID := stored.User.ID
	active, err := s.repo.ConsumeResetAttempt(ctx, userID, maxResetAttempts)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if !hmac.Equal([]byte(active.CodeHash), []byte(s.resetCodeHash(userID, code))) {
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, userID, active.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if s.opts.Sessions != nil {
		s.opts.Sessions.DisconnectUser(userID)
	}
	return nil
}

// DeleteAccount permanently deletes the account after re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrWrongPassword
	}
	deleted, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if s.opts.Storage != nil {
		for _, key := range deleted.StorageKeys {
			if err := s.opts.Storage.Delete(ctx, key); err != nil {
				log.Printf(`{"event":"account_delete_file_failed","error":%q}`, err.Error())
			}
		}
	}
	if s.opts.Sessions != nil {
		s.opts.Sessions.DisconnectUser(userID)
	}
	for _, other := range deleted.MatchedUserIDs {
		s.opts.Publisher.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": userID}})
	}
	return nil
}
