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

	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidEmail        = errors.New("invalid email address")
)

// dummyPasswordHash is computed once at start-up with the production cost.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser"), bcrypt.DefaultCost)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	onDeleted func(DeletedAccount)
}

// Option customises the auth service.
type Option func(*Service)

// WithMailer sets the transport used for account recovery e-mails.
func WithMailer(m mailer.Mailer) Option {
	return func(s *Service) { s.mailer = m }
}

// WithDeletionHook runs after an account is erased (file cleanup, events).
func WithDeletionHook(fn func(DeletedAccount)) Option {
	return func(s *Service) { s.onDeleted = fn }
}

func NewService(repo Repository, jwtSecret []byte, opts ...Option) *Service {
	s := &Service{repo: repo, jwtSecret: jwtSecret, mailer: mailer.LogMailer{}}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail := normalizeEmail(email)
	if !ValidEmail(normalizedEmail) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return Tokens{}, User{}, err
	}

	created, err := s.repo.CreateUser(ctx, normalizedEmail, hash)
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
			// Spend the same bcrypt time as a real check so response timing
			// does not reveal which e-mails are registered.
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

// ValidEmail accepts a bare RFC 5322 address (no display name) with a dotted domain.
func ValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" {
		return false
	}
	at := strings.LastIndex(email, "@")
	return at > 0 && strings.Contains(email[at+1:], ".")
}

func toUser(stored StoredUser) User {
	return User{
		ID:             stored.ID,
		Email:          stored.Email,
		OrganizationID: stored.OrganizationID,
		CreatedAt:      stored.CreatedAt,
	}
}
