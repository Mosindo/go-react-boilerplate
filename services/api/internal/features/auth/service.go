package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/mail"
	"net/url"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
	ErrUserNotFound        = errors.New("user not found")
	ErrAccountSuspended    = errors.New("account suspended")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetTokenTTL   = time.Hour

	minPasswordLength = 8
	// bcrypt ignores everything after 72 bytes, so longer passwords are rejected rather than truncated.
	maxPasswordBytes = 72
)

// dummyHash is compared against when an email is unknown so that login timing
// does not reveal which emails are registered.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser"), bcrypt.DefaultCost)

// FileRemover deletes stored photo files once an account is gone.
type FileRemover interface {
	Remove(keys ...string)
}

// SessionEnder closes realtime connections of a user.
type SessionEnder interface {
	Disconnect(userID string)
}

type Service struct {
	repo        Repository
	jwtSecret   []byte
	mailer      mailer.Mailer
	resetURL    string
	files       FileRemover
	connections SessionEnder
}

type Options struct {
	Mailer      mailer.Mailer
	ResetURL    string
	Files       FileRemover
	Connections SessionEnder
}

func NewService(repo Repository, jwtSecret []byte, opts Options) *Service {
	if opts.Mailer == nil {
		opts.Mailer = mailer.LogMailer{}
	}
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: opts.Mailer, resetURL: opts.ResetURL, files: opts.Files, connections: opts.Connections}
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail, err := validateEmail(email)
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
	tokenHash := hashToken(strings.TrimSpace(refreshToken))
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

	refreshSecret, err := generateToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotatedSession, err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(refreshSecret), userAgent, ipAddress, time.Now().Add(refreshTokenTTL), time.Now())
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
	return Tokens{AccessToken: accessToken, RefreshToken: refreshSecret}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(strings.TrimSpace(refreshToken))
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

// ForgotPassword emails a single-use reset link. It never reveals whether the email exists.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalizedEmail := normalizeEmail(email)
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizedEmail)
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

	body := "Bonjour,\n\nPour choisir un nouveau mot de passe, utilisez ce code dans l'application :\n\n" + token +
		"\n\nIl expire dans une heure. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n"
	if s.resetURL != "" {
		if link, err := url.Parse(s.resetURL); err == nil {
			q := link.Query()
			q.Set("token", token)
			link.RawQuery = q.Encode()
			body += "\nLien : " + link.String() + "\n"
		}
	}
	if err := s.mailer.Send(ctx, stored.User.Email, "Réinitialisation de votre mot de passe", body); err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","error":%q}`, err.Error())
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	userID, err := s.repo.ConsumePasswordReset(ctx, hashToken(strings.TrimSpace(token)), string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryResetGone) {
			return ErrInvalidResetToken
		}
		return err
	}
	if s.connections != nil {
		s.connections.Disconnect(userID)
	}
	return nil
}

// DeleteAccount permanently removes the account after re-checking the password.
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
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	if s.files != nil {
		s.files.Remove(keys...)
	}
	if s.connections != nil {
		s.connections.Disconnect(userID)
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := generateToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashToken(refreshToken), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
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

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	if strings.TrimSpace(token) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateEmail(email string) (string, error) {
	normalized := normalizeEmail(email)
	if normalized == "" || len(normalized) > 254 {
		return "", platformerrors.Invalid("email", "email invalide")
	}
	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized || !strings.Contains(normalized[strings.LastIndex(normalized, "@"):], ".") {
		return "", platformerrors.Invalid("email", "email invalide")
	}
	return normalized, nil
}

func validatePassword(password string) error {
	if len([]rune(password)) < minPasswordLength {
		return platformerrors.Invalid("password", "le mot de passe doit contenir au moins 8 caractères")
	}
	if len(password) > maxPasswordBytes {
		return platformerrors.Invalid("password", "le mot de passe est trop long (72 octets maximum)")
	}
	return nil
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, ProfileComplete: stored.ProfileComplete, CreatedAt: stored.CreatedAt}
}
