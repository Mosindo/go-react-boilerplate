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
	ErrInvalidEmail        = errors.New("invalid email")
	ErrAccountSuspended    = errors.New("account suspended")
	ErrPasswordTooLong     = errors.New("password exceeds 72 bytes")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour

	resetCodeTTL         = time.Hour
	resetCodeMaxAttempts = 5
	resetCodeAlphabet    = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	resetCodeLength      = 8
	// maxPasswordBytes is bcrypt's hard input limit.
	maxPasswordBytes = 72
)

// dummyHash lets Login spend the same bcrypt time for unknown emails so
// response timing does not reveal which emails are registered.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
}

func NewService(repo Repository, jwtSecret []byte, mail mailer.Mailer) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		mailer:    mail,
	}
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail := normalizeEmail(email)
	if !validEmail(normalizedEmail) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
	if len(password) > maxPasswordBytes {
		return Tokens{}, User{}, ErrPasswordTooLong
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
	if !validEmail(normalizedEmail) {
		return Tokens{}, User{}, ErrInvalidCredentials
	}

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
	if stored.Suspended {
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

// ForgotPassword emails a short-lived one-time code. It never reveals whether
// the email is registered.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	if !validEmail(normalizeEmail(email)) {
		return ErrInvalidEmail
	}
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
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashRefreshToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := "Votre code de réinitialisation : " + code + "\n\n" +
		"Il est valable une heure. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message."
	if err := s.mailer.Send(ctx, stored.User.Email, "Réinitialisation de votre mot de passe", body); err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","user_id":%q,"error":%q}`, stored.User.ID, err.Error())
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if len(newPassword) > maxPasswordBytes {
		return ErrPasswordTooLong
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	reset, err := s.repo.GetOpenPasswordReset(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if reset.Attempts >= resetCodeMaxAttempts || time.Now().After(reset.ExpiresAt) {
		return ErrInvalidResetCode
	}
	candidate := hashRefreshToken(strings.ToUpper(strings.TrimSpace(code)))
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(reset.CodeHash)) != 1 {
		if err := s.repo.IncrementPasswordResetAttempts(ctx, reset.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumePasswordReset(ctx, reset.ID, stored.User.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

// IsSessionActive lets the HTTP middleware reject revoked sessions.
func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

func generateResetCode() (string, error) {
	buf := make([]byte, resetCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLength)
	for i, b := range buf {
		out[i] = resetCodeAlphabet[int(b)%len(resetCodeAlphabet)]
	}
	return string(out), nil
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

// validEmail reports whether the (already normalized) value is a bare,
// plausible address: one "@", a dotted domain, no display name, <= 254 chars.
func validEmail(email string) bool {
	if len(email) > 254 || strings.ContainsAny(email, " \t\r\n<>,;\"") {
		return false
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return false
	}
	at := strings.LastIndex(email, "@")
	return at > 0 && strings.Contains(email[at+1:], ".") && !strings.HasSuffix(email, ".")
}

func toUser(stored StoredUser) User {
	return User{
		ID:             stored.ID,
		Email:          stored.Email,
		OrganizationID: stored.OrganizationID,
		CreatedAt:      stored.CreatedAt,
	}
}
