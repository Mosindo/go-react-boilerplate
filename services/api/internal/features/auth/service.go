package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"example.com/api/internal/platform/apperr"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = apperr.Conflict("email_exists", "an account already exists for this email")
	ErrInvalidCredentials  = apperr.Unauthorized("invalid_credentials", "invalid email or password")
	ErrInvalidRefreshToken = apperr.Unauthorized("invalid_refresh_token", "session expired, please sign in again")
	ErrUserNotFound        = apperr.Unauthorized("user_not_found", "account not found")
	ErrTermsRequired       = apperr.BadRequest("terms_required", "you must confirm you are 18+ and accept the community rules")
	ErrWeakPassword        = apperr.BadRequest("weak_password", "password must be 8+ characters and contain a letter and a digit")
	ErrInvalidResetCode    = apperr.BadRequest("invalid_reset_code", "invalid or expired code")
	ErrWrongPassword       = apperr.Forbidden("wrong_password", "current password is incorrect")
	ErrInvalidEmail        = apperr.BadRequest("invalid_email", "invalid email address")
)

const (
	accessTokenTTL      = 15 * time.Minute
	refreshTokenTTL     = 30 * 24 * time.Hour
	resetCodeTTL        = 15 * time.Minute
	maxResetAttempts    = 5
	maxResetCodesPerDay = 5
)

// dummyHash is compared against when an e-mail is unknown so that login
// timing does not reveal which accounts exist.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

// Disconnecter closes live realtime connections of a user.
type Disconnecter interface {
	Disconnect(userID string)
}

type Service struct {
	repo       Repository
	jwtSecret  []byte
	mailer     mailer.Mailer
	store      storage.Store
	realtime   Disconnecter
	bcryptCost int
}

type Deps struct {
	Mailer   mailer.Mailer
	Store    storage.Store
	Realtime Disconnecter
	// BcryptCost defaults to bcrypt.DefaultCost; tests may lower it.
	BcryptCost int
}

func NewService(repo Repository, jwtSecret []byte, deps Deps) *Service {
	cost := deps.BcryptCost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &Service{
		repo:       repo,
		jwtSecret:  jwtSecret,
		mailer:     deps.Mailer,
		store:      deps.Store,
		realtime:   deps.Realtime,
		bcryptCost: cost,
	}
}

// ValidatePassword enforces the password policy.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
		return ErrWeakPassword
	}
	var hasLetter, hasDigit bool
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

func (s *Service) Register(ctx context.Context, req RegisterRequest, userAgent, ipAddress string) (Tokens, User, error) {
	if !req.AcceptTerms {
		return Tokens{}, User{}, ErrTermsRequired
	}
	email, err := ValidateEmail(req.Email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := ValidatePassword(req.Password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	user, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, NormalizeEmail(email))
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
	tokens, err := s.issueSessionTokens(ctx, stored.User, userAgent, ipAddress)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, stored.User, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	currentHash := hashToken(strings.TrimSpace(refreshToken))
	if currentHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, currentHash)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrUserNotFound
		}
		return Tokens{}, User{}, err
	}
	nextToken, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, currentHash, hashToken(nextToken), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	accessToken, err := signAccessToken(s.jwtSecret, user.ID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	_ = s.repo.TouchActivity(ctx, user.ID)
	return Tokens{AccessToken: accessToken, RefreshToken: nextToken}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return ErrInvalidRefreshToken
	}
	if _, err := s.repo.RevokeSessionByTokenHash(ctx, tokenHash); err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	_ = s.repo.TouchActivity(ctx, userID)
	return user, nil
}

func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// ChangePassword keeps the current session and revokes every other one.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, currentPassword, newPassword string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(currentPassword)) != nil {
		return ErrWrongPassword
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	return s.repo.RevokeUserSessions(ctx, userID, sessionID)
}

// RequestPasswordReset e-mails a 6-digit code. It never reveals whether the
// address has an account.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalized := NormalizeEmail(email)
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	if stored.User.IsDemo {
		return nil
	}
	recent, err := s.repo.CountRecentResetCodes(ctx, stored.User.ID, time.Now().Add(-24*time.Hour))
	if err != nil {
		return err
	}
	if recent >= maxResetCodesPerDay {
		return nil
	}
	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreateResetCode(ctx, stored.User.ID, hashToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Votre code de réinitialisation Lueur est : %s\n\nIl expire dans 15 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.", code)
	if err := s.mailer.Send(ctx, stored.User.Email, "Lueur — code de réinitialisation", body); err != nil {
		log.Printf(`{"event":"password_reset_mail_error","error":%q}`, err.Error())
	}
	return nil
}

// ResetPassword checks the code (limited attempts), sets the new password and
// revokes every session.
func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	resetCode, err := s.repo.GetActiveResetCode(ctx, stored.User.ID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	if resetCode.Attempts >= maxResetAttempts {
		return ErrInvalidResetCode
	}
	if subtle.ConstantTimeCompare([]byte(resetCode.CodeHash), []byte(hashToken(strings.TrimSpace(code)))) != 1 {
		if err := s.repo.IncrementResetAttempts(ctx, resetCode.ID); err != nil {
			return err
		}
		return ErrInvalidResetCode
	}
	if err := s.repo.ConsumeResetCode(ctx, resetCode.ID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, stored.User.ID, string(hash)); err != nil {
		return err
	}
	if err := s.repo.RevokeUserSessions(ctx, stored.User.ID, ""); err != nil {
		return err
	}
	if s.realtime != nil {
		s.realtime.Disconnect(stored.User.ID)
	}
	return nil
}

// DeleteAccount permanently erases the user and all personal data after a
// password confirmation, then removes stored photo files.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return ErrWrongPassword
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	if s.store != nil {
		for _, key := range keys {
			if err := s.store.Delete(ctx, key); err != nil {
				log.Printf(`{"event":"photo_file_delete_error","key":%q,"error":%q}`, key, err.Error())
			}
		}
	}
	if s.realtime != nil {
		s.realtime.Disconnect(userID)
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashToken(refreshToken), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := signAccessToken(s.jwtSecret, user.ID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	_ = s.repo.TouchActivity(ctx, user.ID)
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

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomDigits(n int) (string, error) {
	var b strings.Builder
	for i := 0; i < n; i++ {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + d.Int64()))
	}
	return b.String(), nil
}

func hashToken(token string) string {
	if strings.TrimSpace(token) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ValidateEmail normalises then validates an address (single mailbox, no
// display name, a dotted domain).
func ValidateEmail(raw string) (string, error) {
	email := NormalizeEmail(raw)
	if len(email) < 3 || len(email) > 254 {
		return "", ErrInvalidEmail
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || parsed.Name != "" {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndexByte(email, '@')
	if at < 1 || !strings.Contains(email[at+1:], ".") || strings.HasSuffix(email, ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// NormalizeEmail trims and lower-cases an e-mail address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
