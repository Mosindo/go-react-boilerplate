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
	"log"
	netmail "net/mail"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mail"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetCodeTTL    = 30 * time.Minute
	minPasswordLen  = 8
	maxPasswordLen  = 72 // bcrypt silently/explicitly rejects longer inputs
	resetAlphabet   = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	resetCodeLen    = 8
)

var (
	errInvalidCredentials = platformerrors.Unauthorized("invalid_credentials", "invalid email or password")
	errInvalidRefresh     = platformerrors.Unauthorized("invalid_refresh_token", "your session has expired, please sign in again")
	errEmailExists        = platformerrors.Conflict("email_taken", "an account with this email already exists")
	errInvalidResetCode   = platformerrors.BadRequest("invalid_reset_code", "the code is invalid or has expired")
	errUserGone           = platformerrors.Unauthorized("user_not_found", "account not found")
)

// FileRemover deletes stored photo blobs once their account is gone.
type FileRemover interface {
	Remove(key string) error
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mail.Mailer
	files     FileRemover
	// dummyHash lets Login spend the same bcrypt time for unknown emails (no timing oracle).
	dummyHash []byte
}

func NewService(repo Repository, jwtSecret []byte, mailer mail.Mailer, files FileRemover) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: mailer, files: files, dummyHash: dummy}
}

func validatePassword(password string) error {
	if len(password) < minPasswordLen {
		return platformerrors.BadRequest("weak_password", "password must be at least 8 characters")
	}
	if len(password) > maxPasswordLen {
		return platformerrors.BadRequest("weak_password", "password must be at most 72 bytes")
	}
	return nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalized := normalizeEmail(email)
	if err := validateEmail(normalized); err != nil {
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
			return Tokens{}, User{}, errEmailExists
		}
		return Tokens{}, User{}, err
	}
	user := toUser(created)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ipAddress)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Tokens{}, User{}, errInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, errInvalidCredentials
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
	stored, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, errUserGone
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
			return Tokens{}, User{}, errInvalidRefresh
		}
		return Tokens{}, User{}, err
	}
	user := toUser(stored)
	access, err := signAccessToken(s.jwtSecret, user.ID, rotated.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	err := s.repo.RevokeSessionByTokenHash(ctx, hashRefreshToken(refreshToken), time.Now())
	if err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return User{}, errUserGone
		}
		return User{}, err
	}
	return toUser(stored), nil
}

// ChangePassword verifies the current password, then revokes all sessions (the client signs in again).
func (s *Service) ChangePassword(ctx context.Context, userID, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	if err := s.verifyPassword(ctx, userID, current); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash))
}

// ForgotPassword always succeeds from the caller's point of view; mail is sent asynchronously
// so response time does not reveal whether the account exists.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized := normalizeEmail(email)
	if err := validateEmail(normalized); err != nil {
		return err
	}
	code, err := generateResetCode()
	if err != nil {
		return err
	}
	exists, err := s.repo.CreatePasswordReset(ctx, normalized, s.hashResetCode(normalized, code), time.Now().Add(resetCodeTTL))
	if err != nil {
		return err
	}
	if exists {
		go func() {
			body := "Your recovery code is: " + code + "\n\nIt expires in 30 minutes. If you did not ask for it, ignore this message."
			if err := s.mailer.Send(normalized, "Your account recovery code", body); err != nil {
				log.Printf(`{"event":"mail_error","operation":"auth.forgot_password","error":%q}`, err.Error())
			}
		}()
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	normalized := normalizeEmail(email)
	err = s.repo.ConsumePasswordReset(ctx, normalized, s.hashResetCode(normalized, normalizeCode(code)), string(hash))
	if errors.Is(err, ErrRepositoryResetBad) {
		return errInvalidResetCode
	}
	return err
}

// DeleteAccount permanently removes the user and all their data, then the photo files.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	if err := s.verifyPassword(ctx, userID, password); err != nil {
		return err
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return errUserGone
		}
		return err
	}
	for _, key := range keys {
		if err := s.files.Remove(key); err != nil {
			log.Printf(`{"event":"file_cleanup_error","operation":"auth.delete_account","error":%q}`, err.Error())
		}
	}
	return nil
}

func (s *Service) verifyPassword(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return errUserGone
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return platformerrors.Forbidden("wrong_password", "the password is incorrect")
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ipAddress string) (Tokens, error) {
	refresh, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashRefreshToken(refresh), userAgent, ipAddress, time.Now().Add(refreshTokenTTL))
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
	claims := jwt.MapClaims{
		"uid": userID,
		"sid": sessionID,
		"iat": jwt.NewNumericDate(now),
		"exp": jwt.NewNumericDate(now.Add(accessTokenTTL)),
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

func hashRefreshToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateResetCode() (string, error) {
	buf := make([]byte, resetCodeLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLen)
	for i, b := range buf {
		out[i] = resetAlphabet[int(b)%len(resetAlphabet)] // 256 % 32 == 0: unbiased
	}
	return string(out), nil
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
}

// hashResetCode binds the code to the email and the server secret.
func (s *Service) hashResetCode(email, code string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte(email + "|" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validateEmail checks the already-normalised address: a bare "local@domain.tld", at most 254 bytes.
func validateEmail(email string) error {
	addr, err := netmail.ParseAddress(email)
	at := strings.LastIndex(email, "@")
	if err != nil || addr.Address != email || addr.Name != "" || len(email) > 254 ||
		at < 1 || !strings.Contains(email[at+1:], ".") {
		return platformerrors.BadRequest("invalid_email", "enter a valid email address")
	}
	return nil
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, CreatedAt: stored.CreatedAt, ProfileComplete: stored.ProfileComplete}
}
