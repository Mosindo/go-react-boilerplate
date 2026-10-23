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
	"log"
	"net/mail"
	"strings"
	"time"
	"unicode"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetCodeTTL    = 30 * time.Minute
	maxResetTries   = 5

	minPasswordLen = 8
	// bcrypt silently truncates after 72 bytes, so longer passwords are refused.
	maxPasswordBytes = 72

	// Unambiguous characters (no 0/O/1/I) so codes are easy to type.
	resetCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	resetCodeLength   = 10
)

var (
	errInvalidCredentials = platformerrors.Unauthorized("invalid_credentials", "invalid credentials")
	errInvalidRefresh     = platformerrors.Unauthorized("invalid_refresh_token", "invalid refresh token")
	errInvalidResetCode   = platformerrors.BadRequest("invalid_reset_code", "invalid or expired code")
	errSuspended          = platformerrors.Forbidden("account_suspended", "account suspended")
)

// dummyHash lets Login spend the same time for unknown emails as for wrong passwords.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	now       func() time.Time
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: m, now: time.Now}
}

// CheckSession is the middleware.SessionChecker backed by this service's repository.
func (s *Service) CheckSession(ctx context.Context, userID, sessionID string) error {
	return s.repo.CheckSession(ctx, userID, sessionID)
}

// AuthenticateToken validates an access token (used by the WebSocket handshake).
func (s *Service) AuthenticateToken(ctx context.Context, token string) (string, error) {
	id, err := middleware.ParseAccessToken(ctx, s.jwtSecret, token, s.repo.CheckSession)
	if err != nil {
		return "", err
	}
	return id.UserID, nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalized, err := validateEmail(email)
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
	created, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, platformerrors.Conflict("email_taken", "email already exists")
		}
		return Tokens{}, User{}, err
	}
	user := toUser(created)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalized := normalizeEmail(email)
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return Tokens{}, User{}, errInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return Tokens{}, User{}, errInvalidCredentials
	}
	if stored.User.Status != "active" {
		return Tokens{}, User{}, errSuspended
	}
	user := toUser(stored.User)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	currentHash := hashRefreshToken(refreshToken)
	if currentHash == "" {
		return Tokens{}, User{}, errInvalidRefresh
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, currentHash)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, errInvalidRefresh
		}
		return Tokens{}, User{}, err
	}
	userRecord, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, errInvalidRefresh
		}
		return Tokens{}, User{}, err
	}
	if userRecord.Status != "active" {
		return Tokens{}, User{}, errSuspended
	}
	next, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	now := s.now()
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, currentHash, hashRefreshToken(next), userAgent, ipAddress, now.Add(refreshTokenTTL), now)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, errInvalidRefresh
		}
		return Tokens{}, User{}, err
	}
	user := toUser(userRecord)
	access, err := s.signAccessToken(user.ID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashRefreshToken(refreshToken)
	if tokenHash == "" {
		return platformerrors.BadRequest("invalid_refresh_token", "invalid refresh token")
	}
	err := s.repo.RevokeSessionByTokenHash(ctx, tokenHash, s.now())
	if err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, OnboardingState, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, OnboardingState{}, platformerrors.Unauthorized("invalid_token", "user not found")
		}
		return User{}, OnboardingState{}, err
	}
	state, err := s.repo.OnboardingState(ctx, userID)
	return toUser(stored), state, err
}

// DeleteAccount permanently removes the user after a password confirmation.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return platformerrors.Unauthorized("invalid_token", "user not found")
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return platformerrors.Forbidden("invalid_password", "password is incorrect")
	}
	return s.repo.DeleteUser(ctx, userID)
}

// ForgotPassword never reveals whether the email is registered.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized := normalizeEmail(email)
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	if stored.User.Status != "active" {
		return nil
	}
	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, s.hashResetCode(stored.User.ID, code), s.now().Add(resetCodeTTL)); err != nil {
		return err
	}
	// Sent in the background so response time does not reveal whether the account exists.
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		body := fmt.Sprintf("Votre code de réinitialisation : %s\n\nIl expire dans 30 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.", code)
		if err := s.mailer.Send(sendCtx, normalized, "Réinitialisation de votre mot de passe", body); err != nil {
			log.Printf(`{"event":"reset_mail_failed","error":%q}`, err.Error())
		}
	}()
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return errInvalidResetCode
		}
		return err
	}
	reset, err := s.repo.GetLatestPasswordReset(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryResetGone) {
			return errInvalidResetCode
		}
		return err
	}
	if reset.Attempts >= maxResetTries {
		return errInvalidResetCode
	}
	candidate := s.hashResetCode(stored.User.ID, strings.ToUpper(strings.TrimSpace(code)))
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(reset.CodeHash)) != 1 {
		if err := s.repo.RegisterResetAttempt(ctx, reset.ID); err != nil {
			return err
		}
		return errInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumePasswordReset(ctx, reset.ID, stored.User.ID, string(hash), s.now()); err != nil {
		if errors.Is(err, ErrRepositoryResetGone) {
			return errInvalidResetCode
		}
		return err
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refresh, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashRefreshToken(refresh), userAgent, ipAddress, s.now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := s.signAccessToken(user.ID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *Service) signAccessToken(userID, sessionID string) (string, error) {
	now := s.now()
	claims := jwt.MapClaims{
		"uid": userID,
		"sid": sessionID,
		"iss": middleware.TokenIssuer,
		"iat": now.Unix(),
		"exp": now.Add(accessTokenTTL).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte("password-reset|" + userID + "|" + code))
	return hex.EncodeToString(mac.Sum(nil))
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
		out[i] = resetCodeAlphabet[int(v)%len(resetCodeAlphabet)] // 256 % 32 == 0: no modulo bias
	}
	return string(out), nil
}

func hashRefreshToken(refreshToken string) string {
	trimmed := strings.TrimSpace(refreshToken)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateEmail(email string) (string, error) {
	normalized := normalizeEmail(email)
	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized || len(normalized) > 254 || !strings.Contains(normalized[strings.LastIndex(normalized, "@"):], ".") {
		return "", platformerrors.Validation(map[string]string{"email": "invalid email address"})
	}
	return normalized, nil
}

func validatePassword(password string) error {
	var hasLetter, hasDigit bool
	for _, r := range password {
		hasLetter = hasLetter || unicode.IsLetter(r)
		hasDigit = hasDigit || unicode.IsDigit(r)
	}
	switch {
	case len(password) < minPasswordLen:
		return platformerrors.Validation(map[string]string{"password": "must be at least 8 characters"})
	case len(password) > maxPasswordBytes:
		return platformerrors.Validation(map[string]string{"password": "must be at most 72 bytes"})
	case !hasLetter || !hasDigit:
		return platformerrors.Validation(map[string]string{"password": "must contain a letter and a digit"})
	}
	return nil
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, CreatedAt: stored.CreatedAt}
}
