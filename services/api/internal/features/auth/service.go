package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
	ErrInvalidEmail        = errors.New("invalid email address")
	ErrWeakPassword        = errors.New("password must be 8-72 characters with at least one letter and one digit")
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 30 * 24 * time.Hour
	passwordResetTTL = time.Hour
)

// Mailer delivers account-recovery messages.
type Mailer interface {
	SendPasswordReset(ctx context.Context, toEmail, token string) error
}

// dummyHash is a valid bcrypt hash of a random value, used to equalise timing.
var dummyHash = func() string {
	h, err := bcrypt.GenerateFromPassword([]byte("timing-equaliser-"+time.Now().String()), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}()

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    Mailer
}

func NewService(repo Repository, jwtSecret []byte) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// WithMailer enables password recovery emails.
func (s *Service) WithMailer(m Mailer) *Service {
	s.mailer = m
	return s
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail := normalizeEmail(email)
	if !validEmail(normalizedEmail) {
		return Tokens{}, User{}, ErrInvalidEmail
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
			// Same work as a real check, so response time does not reveal
			// whether the e-mail is registered.
			_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
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

// ForgotPassword always succeeds from the caller's point of view so that the
// endpoint cannot be used to discover which emails are registered.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	if s.mailer == nil {
		return nil
	}
	token, err := generateRefreshToken()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashRefreshToken(token), time.Now().Add(passwordResetTTL)); err != nil {
		return err
	}
	return s.mailer.SendPasswordReset(ctx, stored.User.Email, token)
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumePasswordReset(ctx, hashRefreshToken(strings.TrimSpace(token)), string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}
	return nil
}

// DeleteAccount permanently removes the user and every associated row after
// re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	full, err := s.repo.GetUserAuthByEmail(ctx, stored.Email)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(full.PasswordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return s.repo.DeleteUser(ctx, userID)
}

func validatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return ErrWeakPassword
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127:
			hasLetter = true
		}
	}
	if !hasLetter || !hasDigit {
		return ErrWeakPassword
	}
	return nil
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
		UserID:    user.ID,
		SessionID: sessionID,
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

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email && strings.Contains(email[strings.LastIndex(email, "@"):], ".")
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func toUser(stored StoredUser) User {
	return User{
		ID:        stored.ID,
		Email:     stored.Email,
		CreatedAt: stored.CreatedAt,
	}
}
