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
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = platformerrors.Conflict("email already exists")
	ErrInvalidCredentials  = platformerrors.Unauthorized("invalid credentials")
	ErrInvalidRefreshToken = platformerrors.Unauthorized("invalid refresh token")
	ErrUserNotFound        = platformerrors.Unauthorized("user not found")
	ErrInvalidResetCode    = platformerrors.BadRequest("invalid or expired recovery code")
	errInvalidEmail        = platformerrors.BadRequest("invalid email address")
	errPasswordTooLong     = platformerrors.BadRequest("password must be at most 72 bytes")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetCodeTTL    = 15 * time.Minute
	resetAlphabet   = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I
	resetCodeLength = 10
)

// dummyHash lets Login spend the same bcrypt time for unknown emails.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

// ConnectionCloser drops live realtime connections of a user.
type ConnectionCloser interface {
	DisconnectUser(userID string)
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	files     storage.Store
	conns     ConnectionCloser
	appName   string
}

type Options struct {
	Mailer  mailer.Mailer
	Files   storage.Store
	Conns   ConnectionCloser
	AppName string
}

func NewService(repo Repository, jwtSecret []byte, opts Options) *Service {
	if opts.Mailer == nil {
		opts.Mailer = mailer.Discard{}
	}
	if opts.AppName == "" {
		opts.AppName = "Lumen"
	}
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: opts.Mailer, files: opts.Files, conns: opts.Conns, appName: opts.AppName}
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	if len([]byte(password)) > 72 {
		return Tokens{}, User{}, errPasswordTooLong
	}
	normalized, err := parseEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}

	created, err := s.repo.CreateUser(ctx, normalized, string(hash))
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

	user := toUser(stored.User)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	_ = s.repo.TouchActivity(ctx, user.ID)
	return tokens, user, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	currentHash := hashSecret(strings.TrimSpace(refreshToken))
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

	refreshSecret, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, currentHash, hashSecret(refreshSecret),
		userAgent, ipAddress, time.Now().Add(refreshTokenTTL), time.Now())
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	user := toUser(userRecord)
	accessToken, err := signAccessToken(s.jwtSecret, user.ID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	_ = s.repo.TouchActivity(ctx, user.ID)
	return Tokens{AccessToken: accessToken, RefreshToken: refreshSecret}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	err := s.repo.RevokeSessionByTokenHash(ctx, hashSecret(strings.TrimSpace(refreshToken)), time.Now())
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

// RequestPasswordReset never reveals whether the email exists. Delivery
// failures are logged, not surfaced, for the same reason.
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
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashSecret(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Your %s recovery code is: %s\n\nIt expires in %d minutes. If you did not ask for it, ignore this email.",
		s.appName, code, int(resetCodeTTL.Minutes()))
	if err := s.mailer.Send(ctx, stored.User.Email, s.appName+" account recovery", body); err != nil {
		log.Printf("password reset email failed: %v", err)
	}
	return nil
}

func (s *Service) ConfirmPasswordReset(ctx context.Context, code, newPassword string) error {
	if len([]byte(newPassword)) > 72 {
		return errPasswordTooLong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := s.repo.ConsumePasswordReset(ctx, hashSecret(strings.ToUpper(strings.TrimSpace(code))), string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

// DeleteAccount erases the user after a password check, then removes the
// photo files and closes live connections.
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

	keys, err := s.repo.ListPhotoKeys(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}
	if s.files != nil {
		for _, key := range keys {
			if err := s.files.Delete(key); err != nil {
				log.Printf("delete photo file %s: %v", key, err)
			}
		}
	}
	if s.conns != nil {
		s.conns.DisconnectUser(userID)
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashSecret(refreshToken), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := signAccessToken(s.jwtSecret, user.ID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
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

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateResetCode() (string, error) {
	b := make([]byte, resetCodeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLength)
	for i, v := range b {
		out[i] = resetAlphabet[int(v)%len(resetAlphabet)] // 256 % 32 == 0: no modulo bias
	}
	return string(out), nil
}

func hashSecret(secret string) string {
	if strings.TrimSpace(secret) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// parseEmail trims, lowercases and validates an address (plain addr-spec only).
func parseEmail(raw string) (string, error) {
	email := normalizeEmail(raw)
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || len(email) > 254 {
		return "", errInvalidEmail
	}
	at := strings.LastIndexByte(email, '@')
	if at < 1 || !strings.Contains(email[at+1:], ".") {
		return "", errInvalidEmail
	}
	return email, nil
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, CreatedAt: stored.CreatedAt, ProfileComplete: stored.ProfileComplete}
}
