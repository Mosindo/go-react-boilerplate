package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	netmail "net/mail"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/middleware"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = platformerrors.New(http.StatusConflict, "email_taken", "an account with this email already exists")
	ErrInvalidCredentials  = platformerrors.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	ErrInvalidRefreshToken = platformerrors.New(http.StatusUnauthorized, "unauthorized", "invalid refresh token")
	ErrUserNotFound        = platformerrors.New(http.StatusUnauthorized, "unauthorized", "user not found")
	ErrInvalidResetToken   = platformerrors.New(http.StatusBadRequest, "invalid_token", "this reset link is invalid or has expired")
	ErrInvalidEmail        = platformerrors.New(http.StatusBadRequest, "invalid_request", "email address is not valid")
	ErrWeakPassword        = platformerrors.New(http.StatusBadRequest, "invalid_request", "password must be 8 to 72 bytes")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetTokenTTL   = time.Hour
	maxPasswordLen  = 72 // bcrypt silently truncates beyond this
)

type Service struct {
	repo        Repository
	jwtSecret   []byte
	mailer      mail.Mailer
	appLinkBase string
	dummyHash   []byte
}

func NewService(repo Repository, jwtSecret []byte, mailer mail.Mailer, appLinkBase string) *Service {
	// Compared against when the e-mail is unknown so login timing does not reveal registered addresses.
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equaliser"), bcrypt.DefaultCost)
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: mailer, appLinkBase: appLinkBase, dummyHash: dummy}
}

func validPassword(p string) bool { return len(p) >= 8 && len(p) <= maxPasswordLen }

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	email = normalizeEmail(email)
	if !validEmail(email) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
	if !validPassword(password) {
		return Tokens{}, User{}, ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	created, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	user := toUser(created)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	user := toUser(stored.User)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, hashToken(strings.TrimSpace(refreshToken)))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByID(ctx, session.UserID)
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
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(next), userAgent, ip, time.Now().Add(refreshTokenTTL), time.Now())
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	access, err := signAccessToken(s.jwtSecret, session.UserID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, toUser(stored), nil
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
	return toUser(stored), nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, current, next string) error {
	if !validPassword(next) {
		return ErrWeakPassword
	}
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return mapNotFound(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	return s.repo.RevokeAllSessions(ctx, userID, time.Now())
}

// DeleteAccount permanently removes the user and all associated data after re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return mapNotFound(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	return s.repo.DeleteUser(ctx, userID)
}

// ForgotPassword always succeeds from the caller's point of view (no account enumeration).
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
	body := fmt.Sprintf("Bonjour,\n\nPour choisir un nouveau mot de passe, ouvrez : %sreset-password?token=%s\n"+
		"ou saisissez ce code dans l'application : %s\n\nCe code expire dans 1 heure. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n",
		s.appLinkBase, token, token)
	return s.mailer.Send(ctx, stored.User.Email, "Réinitialisation de votre mot de passe Ember", body)
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if !validPassword(password) {
		return ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := s.repo.ConsumePasswordReset(ctx, hashToken(strings.TrimSpace(token)), string(hash)); err != nil {
		return mapResetErr(err)
	}
	return nil
}

func mapResetErr(err error) error {
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrInvalidResetToken
	}
	return err
}

func mapNotFound(err error) error {
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrUserNotFound
	}
	return err
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ip string) (Tokens, error) {
	refresh, err := randomToken()
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
	claims := AccessTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    middleware.TokenIssuer,
			Subject:   userID,
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

// validEmail accepts a bare address ("name@host.tld"), not display-name forms like "Bob <b@x.y>".
func validEmail(email string) bool {
	addr, err := netmail.ParseAddress(email)
	return err == nil && addr.Address == email && strings.Contains(email[strings.LastIndex(email, "@"):], ".")
}

func toUser(s StoredUser) User {
	return User{ID: s.ID, Email: s.Email, CreatedAt: s.CreatedAt, ProfileComplete: s.ProfileComplete}
}
