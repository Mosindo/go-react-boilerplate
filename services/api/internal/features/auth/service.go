package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/validate"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = platformerrors.Conflict("email already exists")
	ErrInvalidCredentials  = platformerrors.New(401, "invalid credentials")
	ErrInvalidRefreshToken = platformerrors.New(401, "invalid refresh token")
	ErrUserNotFound        = platformerrors.New(404, "user not found")
	ErrInvalidResetCode    = platformerrors.Invalid("invalid or expired code")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetTokenTTL   = time.Hour
	minPasswordLen  = 8
	maxPasswordLen  = 72 // bcrypt ignores/rejects anything beyond 72 bytes
	resetAlphabet   = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	resetCodeLen    = 10
)

// FileRemover deletes stored files (account deletion removes the user's photos).
type FileRemover interface {
	Delete(ctx context.Context, key string) error
}

// Disconnector drops live realtime connections of a user.
type Disconnector interface {
	Disconnect(userID string)
}

type Service struct {
	repo         Repository
	jwtSecret    []byte
	mail         mailer.Mailer
	files        FileRemover
	disconnector Disconnector
	now          func() time.Time
	dummyHash    []byte
}

func NewService(repo Repository, jwtSecret []byte, mail mailer.Mailer, files FileRemover, disc Disconnector) *Service {
	// Used to keep login timing similar whether or not the e-mail exists.
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)
	return &Service{repo: repo, jwtSecret: jwtSecret, mail: mail, files: files, disconnector: disc, now: time.Now, dummyHash: dummy}
}

func (s *Service) SessionActive(ctx context.Context, sessionID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID)
}

func validatePassword(p string) error {
	if len(p) < minPasswordLen {
		return platformerrors.Invalid("password must be at least 8 characters")
	}
	if len(p) > maxPasswordLen {
		return platformerrors.Invalid("password must be at most 72 bytes")
	}
	return nil
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, userAgent, ip string) (Tokens, User, error) {
	email := validate.NormalizeEmail(req.Email)
	if !validate.ValidEmail(email) {
		return Tokens{}, User{}, platformerrors.Invalid("invalid email")
	}
	if err := validatePassword(req.Password); err != nil {
		return Tokens{}, User{}, err
	}
	firstName, ok := validate.CleanText(req.FirstName, 40, false)
	if !ok || firstName == "" {
		return Tokens{}, User{}, platformerrors.Invalid("invalid first name")
	}
	birth, err := validate.ParseBirthDate(req.BirthDate)
	if err != nil {
		return Tokens{}, User{}, platformerrors.Invalid("invalid birth date (expected YYYY-MM-DD)")
	}
	age := validate.Age(birth, s.now())
	if age < validate.MinAge {
		return Tokens{}, User{}, platformerrors.New(403, "you must be at least 18 years old")
	}
	if age > validate.MaxAge {
		return Tokens{}, User{}, platformerrors.Invalid("invalid birth date")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	created, err := s.repo.CreateAccount(ctx, email, string(hash), firstName, birth)
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	user := User(created)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, validate.NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if len(password) > maxPasswordLen || bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	user := User(stored.StoredUser)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, hashToken(strings.TrimSpace(refreshToken)))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	next, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, hashToken(next), userAgent, ip, s.now().Add(refreshTokenTTL), s.now())
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, ErrInvalidRefreshToken
		}
		return Tokens{}, User{}, err
	}
	user := User(stored)
	access, err := s.signAccessToken(user, rotated.ID)
	return Tokens{AccessToken: access, RefreshToken: next}, user, err
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	h := hashToken(strings.TrimSpace(refreshToken))
	if h == "" {
		return ErrInvalidRefreshToken
	}
	if err := s.repo.RevokeSessionByTokenHash(ctx, h, s.now()); err != nil && !errors.Is(err, ErrRepositorySessionGone) {
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
	return User(stored), nil
}

// ForgotPassword always succeeds from the caller's point of view so it cannot be used to
// discover which e-mails are registered.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = validate.NormalizeEmail(email)
	if !validate.ValidEmail(email) {
		return platformerrors.Invalid("invalid email")
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	code, err := randomCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreateResetToken(ctx, stored.ID, hashToken(code), s.now().Add(resetTokenTTL)); err != nil {
		return err
	}
	body := "Bonjour,\n\nVoici votre code de récupération : " + code +
		"\n\nIl est valable 1 heure. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n"
	if err := s.mail.Send(ctx, stored.Email, "Votre code de récupération", body); err != nil {
		log.Printf(`{"event":"mail_error","error":%q}`, err.Error())
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, code, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != resetCodeLen {
		return ErrInvalidResetCode
	}
	userID, err := s.repo.ConsumeResetToken(ctx, hashToken(code))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	return s.repo.RevokeAllSessions(ctx, userID, s.now())
}

func (s *Service) ChangePassword(ctx context.Context, userID, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	return nil
}

// DeleteAccount permanently removes the account and everything attached to it.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	keys, err := s.repo.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := s.files.Delete(ctx, k); err != nil {
			log.Printf(`{"event":"file_delete_error","key":%q,"error":%q}`, k, err.Error())
		}
	}
	if s.disconnector != nil {
		s.disconnector.Disconnect(userID)
	}
	return nil
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ip string) (Tokens, error) {
	refresh, err := randomToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashToken(refresh), userAgent, ip, s.now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := s.signAccessToken(user, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *Service) signAccessToken(user User, sessionID string) (string, error) {
	now := s.now()
	claims := AccessTokenClaims{
		UserID:    user.ID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomCode() (string, error) {
	b := make([]byte, resetCodeLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, resetCodeLen)
	for i, v := range b {
		out[i] = resetAlphabet[int(v)%len(resetAlphabet)]
	}
	return string(out), nil
}

func hashToken(token string) string {
	if strings.TrimSpace(token) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
