package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/url"
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
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
	ErrUserNotFound        = errors.New("user not found")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetTokenTTL   = time.Hour
)

// dummyHash lets Login spend the same bcrypt time for unknown emails (no timing oracle).
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

// FileDeleter removes stored files (photos) when an account is deleted.
type FileDeleter interface{ Delete(key string) }

type Deps struct {
	Mailer     mailer.Mailer
	Files      FileDeleter
	AppBaseURL string
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	deps      Deps
}

func NewService(repo Repository, jwtSecret []byte, deps Deps) *Service {
	if deps.Mailer == nil {
		deps.Mailer = mailer.LogMailer{}
	}
	return &Service{repo: repo, jwtSecret: jwtSecret, deps: deps}
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
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
	return tokens, user, err
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
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, hashToken(strings.TrimSpace(refreshToken)))
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
	next, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(strings.TrimSpace(refreshToken)), hashToken(next), userAgent, ipAddress, time.Now().Add(refreshTokenTTL), time.Now())
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	user := toUser(userRecord)
	access, err := signAccessToken(s.jwtSecret, user, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	err := s.repo.RevokeSessionByTokenHash(ctx, hashToken(strings.TrimSpace(refreshToken)), time.Now())
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
	_ = s.repo.TouchUser(ctx, userID)
	return toUser(stored), nil
}

// SessionActive backs the RequireUser middleware.
func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// ForgotPassword never reveals whether the email exists.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	token, err := randomToken()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashToken(token), time.Now().Add(resetTokenTTL)); err != nil {
		return err
	}
	body := "Bonjour,\n\nPour choisir un nouveau mot de passe dans l'application, saisissez ce code (valable 1 heure) :\n\n" + token + "\n"
	if s.deps.AppBaseURL != "" {
		body += "\nOu ouvrez : " + s.deps.AppBaseURL + "/reset-password?token=" + url.QueryEscape(token) + "\n"
	}
	body += "\nSi vous n'êtes pas à l'origine de cette demande, ignorez ce message."
	// Sent asynchronously so response time does not reveal whether the account exists.
	go func(to string) {
		if err := s.deps.Mailer.Send(context.Background(), to, "Réinitialisation de votre mot de passe", body); err != nil {
			log.Printf(`{"event":"mail_error","error":%q}`, err.Error())
		}
	}(stored.User.Email)
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumePasswordReset(ctx, hashToken(strings.TrimSpace(token)), string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

// DeleteAccount permanently removes the user and all associated data, after password confirmation.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	keys, err := s.repo.ListPhotoKeys(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}
	if s.deps.Files != nil {
		for _, k := range keys {
			s.deps.Files.Delete(k)
		}
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refresh, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashToken(refresh), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := signAccessToken(s.jwtSecret, user, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
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

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func toUser(s StoredUser) User { return User{ID: s.ID, Email: s.Email, CreatedAt: s.CreatedAt} }
