package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrWeakPassword        = errors.New("password must contain at least one letter and one digit")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrInvalidEmail        = errors.New("invalid email")
)

const (
	accessTokenTTL       = 15 * time.Minute
	refreshTokenTTL      = 30 * 24 * time.Hour
	resetCodeTTL         = 30 * time.Minute
	resetCodeMaxAttempts = 5
	resetCodeLength      = 8
	// Unambiguous alphabet (no 0/O, 1/I/L) for codes typed by hand.
	resetCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

// AccountDeletionHook lets other features clean resources that live outside
// the database (photo files). Collect runs before the rows are deleted and
// returns a cleanup that runs after a successful deletion.
type AccountDeletionHook interface {
	CollectUserObjects(ctx context.Context, userID string) (func(context.Context), error)
}

type Service struct {
	repo         Repository
	jwtSecret    []byte
	mailer       mailer.Mailer
	deletionHook AccountDeletionHook
	dummyHash    []byte
}

func NewService(repo Repository, jwtSecret []byte) *Service {
	// A real bcrypt hash compared against when the e-mail is unknown, so login
	// timing does not reveal which addresses have an account.
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equaliser-not-a-password"), bcrypt.DefaultCost)
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		mailer:    mailer.DisabledMailer{},
		dummyHash: dummy,
	}
}

// WithMailer configures how password reset codes are delivered.
func (s *Service) WithMailer(m mailer.Mailer) *Service {
	if m != nil {
		s.mailer = m
	}
	return s
}

// WithDeletionHook registers resource cleanup for account deletion.
func (s *Service) WithDeletionHook(h AccountDeletionHook) *Service {
	s.deletionHook = h
	return s
}

// IsSessionActive implements middleware.SessionChecker.
func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

func validatePassword(password string) error {
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit || len(password) < 8 || len(password) > 72 {
		return ErrWeakPassword
	}
	return nil
}

// ValidEmail checks a normalised address: a single "@", no spaces, a dotted
// domain and the RFC 5321 length limit.
func ValidEmail(email string) bool {
	if len(email) < 6 || len(email) > 254 || strings.ContainsAny(email, " \t\r\n<>(),;:\"[]\\") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	at := strings.LastIndex(email, "@")
	domain := email[at+1:]
	return at > 0 && strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail := normalizeEmail(email)
	if !ValidEmail(normalizedEmail) {
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

	return tokens, user, nil
}

// ForgotPassword e-mails a one-time code. It never reveals whether the
// address exists: unknown e-mails succeed silently.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
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
	if err := s.repo.CreatePasswordResetToken(ctx, stored.User.ID, s.hashResetCode(stored.User.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}

	body := fmt.Sprintf("Bonjour,\n\nVoici ton code de réinitialisation Lueur : %s\n\nIl expire dans 30 minutes. Si tu n'es pas à l'origine de cette demande, ignore ce message.\n", code)
	return s.mailer.Send(ctx, stored.User.Email, "Ton code de réinitialisation Lueur", body)
}

// ResetPassword consumes a code, sets the new password and signs out every
// device.
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
	token, err := s.repo.GetActivePasswordResetToken(ctx, stored.User.ID, resetCodeMaxAttempts)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}

	normalizedCode := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
	expected := s.hashResetCode(stored.User.ID, normalizedCode)
	if !hmac.Equal([]byte(expected), []byte(token.CodeHash)) {
		if err := s.repo.IncrementPasswordResetAttempts(ctx, token.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, token.ID, stored.User.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

// DeleteAccount permanently removes the user and all associated data after
// re-checking the password.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	cleanup := func(context.Context) {}
	if s.deletionHook != nil {
		cleanup, err = s.deletionHook.CollectUserObjects(ctx, userID)
		if err != nil {
			return err
		}
	}

	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	cleanup(ctx)
	return nil
}

func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte("password-reset|" + userID + "|" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateResetCode() (string, error) {
	// Rejection sampling keeps every symbol equally likely.
	limit := byte(256 - 256%len(resetCodeAlphabet))
	out := make([]byte, 0, resetCodeLength)
	buf := make([]byte, 16)
	for len(out) < resetCodeLength {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if b < limit && len(out) < resetCodeLength {
				out = append(out, resetCodeAlphabet[int(b)%len(resetCodeAlphabet)])
			}
		}
	}
	return string(out), nil
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

func toUser(stored StoredUser) User {
	return User{
		ID:             stored.ID,
		Email:          stored.Email,
		OrganizationID: stored.OrganizationID,
		CreatedAt:      stored.CreatedAt,
	}
}
