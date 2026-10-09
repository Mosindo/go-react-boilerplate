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
	"unicode/utf8"

	platformmail "example.com/api/internal/platform/mail"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrWeakPassword        = errors.New("password must be 8 to 72 bytes and different from the email")
	ErrInvalidResetCode    = errors.New("invalid or expired code")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour

	minPasswordLength = 8
	maxPasswordBytes  = 72 // bcrypt silently ignores/rejects anything longer

	resetCodeTTL         = 30 * time.Minute
	resetCodeMaxAttempts = 5
	resetCodeMaxPerHour  = 3
	resetCodeAlphabet    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 32 symbols: no 0/O/1/I
	resetCodeLength      = 8
)

// dummyHash lets Login spend the same time for unknown e-mails as for wrong passwords.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("lumo-timing-equalizer"), bcrypt.DefaultCost)

// AccountDeletionHook runs before an account row is deleted (e.g. to purge
// uploaded files and close realtime connections).
type AccountDeletionHook func(ctx context.Context, userID string) error

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    platformmail.Mailer
	hooks     []AccountDeletionHook
}

func NewService(repo Repository, jwtSecret []byte, mailer platformmail.Mailer) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: mailer}
}

// OnAccountDeletion registers a cleanup hook executed before the account is removed.
func (s *Service) OnAccountDeletion(hook AccountDeletionHook) {
	s.hooks = append(s.hooks, hook)
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := validatePassword(password, normalized); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}

	created, err := s.repo.CreateUser(ctx, normalized, string(hash))
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
	normalized, err := normalizeEmail(email)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return Tokens{}, User{}, ErrInvalidCredentials
	}

	stored, err := s.repo.GetUserAuthByEmail(ctx, normalized)
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
	oldHash := hashRefreshToken(strings.TrimSpace(refreshToken))
	if oldHash == "" {
		return Tokens{}, User{}, ErrInvalidRefreshToken
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, oldHash)
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

	newSecret, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, oldHash, hashRefreshToken(newSecret), userAgent, ipAddress, time.Now().Add(refreshTokenTTL), time.Now())
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	user := toUser(userRecord)
	accessToken, err := signAccessToken(s.jwtSecret, user, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: newSecret}, user, nil
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

// ForgotPassword e-mails a one-time code. It never reveals whether the account
// exists: unknown addresses and rate-limited users get the same (nil) result.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil
	}
	user, err := s.repo.GetUserAuthByEmail(ctx, normalized)
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
	err = s.repo.CreateResetCode(ctx, user.User.ID, s.hashResetCode(user.User.ID, code), time.Now().Add(resetCodeTTL), resetCodeMaxPerHour)
	if err != nil {
		if errors.Is(err, ErrRepositoryTooManyCodes) {
			return nil
		}
		return err
	}

	body := fmt.Sprintf("Votre code de réinitialisation Lumo : %s\n\nIl est valable %d minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.", code, int(resetCodeTTL.Minutes()))
	// Delivery happens in the background so response time does not reveal
	// whether the address belongs to an account.
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.mailer.Send(sendCtx, normalized, "Votre code de réinitialisation Lumo", body); err != nil {
			log.Printf(`{"event":"reset_mail_failed","error":%q}`, err.Error())
		}
	}()
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return ErrInvalidResetCode
	}
	if err := validatePassword(newPassword, normalized); err != nil {
		return err
	}
	user, err := s.repo.GetUserAuthByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}

	stored, err := s.repo.UseResetCodeAttempt(ctx, user.User.ID, resetCodeMaxAttempts)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	provided := s.hashResetCode(user.User.ID, strings.ToUpper(strings.TrimSpace(code)))
	if subtle.ConstantTimeCompare([]byte(provided), []byte(stored.CodeHash)) != 1 {
		return ErrInvalidResetCode
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumeResetCode(ctx, stored.ID, user.User.ID, string(hash)); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	return nil
}

// ChangePassword verifies the current password, then revokes every other session.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)); err != nil {
		return ErrInvalidCredentials
	}
	if err := validatePassword(next, stored.User.Email); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

// DeleteAccount permanently removes the user after re-checking their password.
// Foreign keys cascade to profile, photos, swipes, matches, messages, etc.
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
	for _, hook := range s.hooks {
		if err := hook(ctx, userID); err != nil {
			return err
		}
	}
	return s.repo.DeleteUser(ctx, userID)
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashRefreshToken(refreshToken), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := signAccessToken(s.jwtSecret, user, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *Service) hashResetCode(userID, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte("password-reset:" + userID + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
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

func generateRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateResetCode() (string, error) {
	buf := make([]byte, resetCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLength)
	for i, b := range buf {
		out[i] = resetCodeAlphabet[int(b)%len(resetCodeAlphabet)] // 256 % 32 == 0: unbiased
	}
	return string(out), nil
}

func hashRefreshToken(refreshToken string) string {
	if strings.TrimSpace(refreshToken) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(email))
	if trimmed == "" || len(trimmed) > 254 {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed || addr.Name != "" {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndex(trimmed, "@")
	if at < 1 || !strings.Contains(trimmed[at+1:], ".") {
		return "", ErrInvalidEmail
	}
	return trimmed, nil
}

func validatePassword(password, email string) error {
	if utf8.RuneCountInString(password) < minPasswordLength || len(password) > maxPasswordBytes {
		return ErrWeakPassword
	}
	if strings.EqualFold(password, email) {
		return ErrWeakPassword
	}
	return nil
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, CreatedAt: stored.CreatedAt}
}
