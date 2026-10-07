package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/mail"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = platformerrors.Conflict("email already exists")
	ErrInvalidCredentials  = platformerrors.Unauthorized("invalid credentials")
	ErrInvalidRefreshToken = platformerrors.Unauthorized("invalid refresh token")
	ErrUserNotFound        = platformerrors.Unauthorized("user not found")
	ErrInvalidResetToken   = platformerrors.Validation("this reset code is invalid or has expired")
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 30 * 24 * time.Hour
	resetTokenTTL    = time.Hour
	maxPasswordBytes = 72 // bcrypt limit
	resetAlphabet    = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	resetCodeLength  = 12
)

// MediaCleaner removes personal media when an account is deleted.
type MediaCleaner interface {
	KeysForUser(ctx context.Context, userID string) ([]string, error)
	Remove(keys []string)
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Sender
	media     MediaCleaner
	dummyHash []byte
}

func NewService(repo Repository, jwtSecret []byte, sender mailer.Sender, media MediaCleaner) *Service {
	// Used to keep login timing constant when the e-mail is unknown.
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: sender, media: media, dummyHash: dummy}
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return platformerrors.Validation("password must be at least 8 characters")
	}
	if len(password) > maxPasswordBytes {
		return platformerrors.Validation("password must be at most 72 bytes")
	}
	return nil
}

func validateEmail(email string) error {
	addr, err := mail.ParseAddress(normalizeEmail(email))
	if err != nil || addr.Address != normalizeEmail(email) || !strings.Contains(addr.Address, ".") {
		return platformerrors.Validation("enter a valid e-mail address")
	}
	return nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	if err := validateEmail(email); err != nil {
		return Tokens{}, User{}, err
	}
	if err := validatePassword(password); err != nil {
		return Tokens{}, User{}, err
	}
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
	if err != nil {
		return Tokens{}, User{}, err
	}
	return tokens, user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
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

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, User, error) {
	currentHash := hashRefreshToken(strings.TrimSpace(refreshToken))
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

	userRecord, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrUserNotFound
		}
		return Tokens{}, User{}, err
	}

	next, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, currentHash, hashRefreshToken(next), userAgent, ipAddress, time.Now().Add(refreshTokenTTL), time.Now())
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}

	user := toUser(userRecord)
	accessToken, err := signAccessToken(s.jwtSecret, user.ID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashRefreshToken(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return platformerrors.Validation("invalid refresh token")
	}
	if err := s.repo.RevokeSessionByTokenHash(ctx, tokenHash, time.Now()); err != nil && !errors.Is(err, ErrRepositorySessionGone) {
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

// RequestPasswordReset never reveals whether the e-mail exists.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	if err := validateEmail(email); err != nil {
		return err
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
	if err := s.repo.CreateResetToken(ctx, stored.User.ID, hashResetCode(code), time.Now().Add(resetTokenTTL)); err != nil {
		return err
	}
	body := "Use this code to reset your password: " + formatResetCode(code) +
		"\n\nIt expires in 1 hour. If you did not ask for it, you can ignore this e-mail."
	if err := s.mailer.Send(ctx, stored.User.Email, "Your password reset code", body); err != nil {
		// Do not leak delivery failures to the caller (enumeration); log for operators.
		log.Printf(`{"event":"password_reset_mail_failed","error":%q}`, err.Error())
	}
	return nil
}

func (s *Service) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = s.repo.ResetPasswordWithToken(ctx, hashResetCode(normalizeResetCode(token)), string(hash), time.Now())
	if errors.Is(err, ErrRepositoryTokenGone) {
		return ErrInvalidResetToken
	}
	return err
}

// DeleteAccount verifies the password, deletes the user (cascading every personal row) and
// then removes stored photo files.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return platformerrors.Forbidden("password is incorrect")
	}

	var keys []string
	if s.media != nil {
		if keys, err = s.media.KeysForUser(ctx, userID); err != nil {
			return err
		}
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}
	if s.media != nil {
		s.media.Remove(keys)
	}
	return nil
}

func (s *Service) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.SessionActive(ctx, sessionID, userID)
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
	accessToken, err := signAccessToken(s.jwtSecret, user.ID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

type accessClaims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func signAccessToken(secret []byte, userID, sessionID string) (string, error) {
	now := time.Now()
	claims := accessClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
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

func hashRefreshToken(refreshToken string) string {
	if strings.TrimSpace(refreshToken) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(sum[:])
}

func generateResetCode() (string, error) {
	buf := make([]byte, resetCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLength)
	for i, b := range buf {
		out[i] = resetAlphabet[int(b)%len(resetAlphabet)]
	}
	return string(out), nil
}

func formatResetCode(code string) string {
	return code[0:4] + "-" + code[4:8] + "-" + code[8:12]
}

func normalizeResetCode(code string) string {
	code = strings.ToUpper(code)
	return strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' {
			return -1
		}
		return r
	}, code)
}

func hashResetCode(code string) string {
	sum := sha256.Sum256([]byte("reset:" + code))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func toUser(stored StoredUser) User {
	return User{
		ID:              stored.ID,
		Email:           stored.Email,
		CreatedAt:       stored.CreatedAt,
		HasProfile:      stored.HasProfile,
		ProfileComplete: stored.ProfileComplete,
	}
}
