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
	"unicode"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrWeakPassword        = errors.New("password must be 8 to 72 bytes and contain a letter and a digit")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrInvalidEmail        = errors.New("invalid email address")
)

// Mailer delivers password reset codes.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

const (
	accessTokenTTL       = 15 * time.Minute
	refreshTokenTTL      = 30 * 24 * time.Hour
	resetCodeTTL         = 15 * time.Minute
	maxResetAttempts     = 5
	maxResetCodesPerHour = 3
	minPasswordLen       = 8
	maxPasswordBytes     = 72 // bcrypt silently ignores anything beyond 72 bytes
)

// dummyHash is compared against when a login targets an unknown e-mail so the
// response time does not reveal whether an account exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser-password-1"), bcrypt.DefaultCost)

// Disconnector closes realtime connections of a member whose sessions were revoked.
type Disconnector interface {
	Disconnect(userID string)
	DisconnectSession(sessionID string)
}

type Service struct {
	repo         Repository
	jwtSecret    []byte
	mailer       Mailer
	disconnector Disconnector
}

// WithDisconnector makes credential changes also close open WebSockets.
func (s *Service) WithDisconnector(d Disconnector) *Service {
	s.disconnector = d
	return s
}

func (s *Service) disconnect(userID string) {
	if s.disconnector != nil {
		s.disconnector.Disconnect(userID)
	}
}

func NewService(repo Repository, jwtSecret []byte, mailer Mailer) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		mailer:    mailer,
	}
}

// ValidatePassword enforces the password policy shared by every flow.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLen || len(password) > maxPasswordBytes {
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
	if !ValidEmail(normalizedEmail) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
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
	if sessionID != "" && s.disconnector != nil {
		s.disconnector.DisconnectSession(sessionID)
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

// SessionActive implements middleware.SessionChecker.
func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.SessionActive(ctx, sessionID, userID)
}

// ChangePassword requires the current password and revokes other sessions.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, currentPassword, newPassword string) error {
	hash, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	next, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(next), sessionID); err != nil {
		return err
	}
	// The current device reconnects transparently with its still-valid session.
	s.disconnect(userID)
	return nil
}

// RequestPasswordReset e-mails a 6 digit code. It never reveals whether the
// address belongs to an account: unknown addresses return nil too.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalizedEmail := normalizeEmail(email)
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	// Per-account cap: each code allows 5 guesses, so issuing codes without
	// limit would let an attacker brute-force the code space.
	recent, err := s.repo.CountRecentResetCodes(ctx, stored.User.ID, time.Now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= maxResetCodesPerHour {
		return nil
	}
	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if _, err := s.repo.CreateResetCode(ctx, stored.User.ID, s.hashResetCode(stored.User.ID, code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := "Bonjour,\n\nVoici votre code de réinitialisation Lueur : " + code +
		"\nIl expire dans 15 minutes.\n\nSi vous n'êtes pas à l'origine de cette demande, ignorez ce message."
	return s.mailer.Send(ctx, normalizedEmail, "Votre code de réinitialisation", body)
}

// ResetPassword verifies the code (max 5 attempts) and sets a new password.
// Every existing session is revoked.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	userID := stored.User.ID
	resetCode, err := s.repo.GetLatestResetCode(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if resetCode.UsedAt != nil || time.Now().After(resetCode.ExpiresAt) {
		return ErrInvalidResetCode
	}
	allowed, err := s.repo.ConsumeResetAttempt(ctx, resetCode.ID, maxResetAttempts)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrInvalidResetCode
	}
	expected := s.hashResetCode(userID, strings.TrimSpace(code))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(resetCode.CodeHash)) != 1 {
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.CompletePasswordReset(ctx, userID, resetCode.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return ErrInvalidResetCode
		}
		return err
	}
	s.disconnect(userID)
	return nil
}

func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte("password-reset:" + userID + ":" + code))
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

// ValidEmail accepts a bare, already normalised address (no display name).
func ValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 || strings.ContainsAny(email, " \t\r\n<>") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
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
