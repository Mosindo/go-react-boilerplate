package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL    = 15 * time.Minute
	refreshTokenTTL   = 30 * 24 * time.Hour
	resetCodeTTL      = 30 * time.Minute
	maxResetsPerHour  = 3
	resetCodeLength   = 10
	resetCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 32 symbols, no 0/O/1/I ambiguity
	bcryptCost        = 12
	dummyPasswordHash = "$2a$12$m3JwPam1hlU7yl0XRsLn7eRf9A9tPUucwfwUGU4hMr7tFysvSlkLy"
)

var (
	ErrEmailExists        = platformerrors.Conflict("email_exists", "Un compte existe déjà avec cette adresse e-mail.")
	ErrInvalidCredentials = platformerrors.Unauthorized("invalid_credentials", "E-mail ou mot de passe incorrect.")
	ErrInvalidRefresh     = platformerrors.Unauthorized("invalid_refresh_token", "Session expirée. Veuillez vous reconnecter.")
	ErrAccountSuspended   = platformerrors.Forbidden("account_suspended", "Ce compte a été suspendu.")
	ErrInvalidResetCode   = platformerrors.Validation("invalid_reset_code", "Code invalide ou expiré.")
	ErrWeakPassword       = platformerrors.Validation("weak_password", "Le mot de passe doit contenir au moins 8 caractères, dont une lettre et un chiffre.")
	ErrWrongPassword      = platformerrors.Forbidden("wrong_password", "Mot de passe incorrect.")
	ErrTermsNotAccepted   = platformerrors.Validation("terms_not_accepted", "Vous devez accepter les conditions d'utilisation.")
	ErrUserNotFound       = platformerrors.Unauthorized("user_not_found", "Compte introuvable.")
	ErrInvalidEmail       = platformerrors.Validation("invalid_email", "Adresse e-mail invalide.")
)

// PhotoStorage deletes photo bytes after an account is removed.
type PhotoStorage interface {
	Delete(key string) error
}

// SessionTerminator closes live connections (WebSocket) of a user.
type SessionTerminator interface {
	Disconnect(userID string)
}

type Service struct {
	repo       Repository
	jwtSecret  []byte
	mailer     mailer.Mailer
	storage    PhotoStorage
	terminator SessionTerminator
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, storage PhotoStorage, terminator SessionTerminator) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, mailer: m, storage: storage, terminator: terminator}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, userAgent, ip string) (Tokens, User, error) {
	if !req.AcceptedTerms {
		return Tokens{}, User{}, ErrTermsNotAccepted
	}
	email := NormalizeEmail(req.Email)
	if !ValidEmail(email) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
	if err := ValidatePassword(req.Password); err != nil {
		return Tokens{}, User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return Tokens{}, User{}, err
	}
	user, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, errRepoEmailExists) {
			return Tokens{}, User{}, ErrEmailExists
		}
		return Tokens{}, User{}, err
	}
	tokens, err := s.issueSession(ctx, user.ID, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, errRepoNotFound) {
			// Constant-time-ish: still run bcrypt so response timing doesn't reveal accounts.
			_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}
	if stored.ModerationStatus == "suspended" {
		return Tokens{}, User{}, ErrAccountSuspended
	}
	tokens, err := s.issueSession(ctx, stored.ID, userAgent, ip)
	return tokens, stored.User, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	currentHash := hashToken(strings.TrimSpace(refreshToken))
	if currentHash == "" {
		return Tokens{}, User{}, ErrInvalidRefresh
	}
	next, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	session, err := s.repo.RotateSession(ctx, currentHash, hashToken(next), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, errRepoNotFound) {
			return Tokens{}, User{}, ErrInvalidRefresh
		}
		return Tokens{}, User{}, err
	}
	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	access, err := signAccessToken(s.jwtSecret, session.UserID, session.ID)
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, user.User, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(strings.TrimSpace(refreshToken))
	if tokenHash == "" {
		return nil
	}
	return s.repo.RevokeSessionByTokenHash(ctx, tokenHash)
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errRepoNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return stored.User, nil
}

// ValidateSession implements middleware.SessionValidator.
func (s *Service) ValidateSession(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.ValidateSession(ctx, sessionID, userID)
}

// ForgotPassword always succeeds from the caller's perspective so it cannot be used to
// discover which e-mail addresses have an account.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	stored, err := s.repo.GetUserByEmail(ctx, NormalizeEmail(email))
	if errors.Is(err, errRepoNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	recent, err := s.repo.CountRecentResetTokens(ctx, stored.ID, time.Now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if recent >= maxResetsPerHour {
		return nil
	}
	code, err := generateResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreateResetToken(ctx, stored.ID, hashToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return err
	}
	body := fmt.Sprintf("Bonjour,\n\nVoici votre code de réinitialisation : %s\n\nIl est valable 30 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n", code)
	if err := s.mailer.Send(stored.Email, "Votre code de réinitialisation Lueur", body); err != nil {
		log.Printf(`{"event":"password_reset_mail_failed","error":%q}`, err.Error())
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	normalizedCode := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	if len(normalizedCode) != resetCodeLength {
		return ErrInvalidResetCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return err
	}
	userID, err := s.repo.ConsumeResetToken(ctx, NormalizeEmail(email), hashToken(normalizedCode), string(hash))
	if err != nil {
		if errors.Is(err, errRepoNotFound) {
			return ErrInvalidResetCode
		}
		return err
	}
	s.terminator.Disconnect(userID)
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(current)) != nil {
		return ErrWrongPassword
	}
	if err := ValidatePassword(next); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcryptCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

// DeleteAccount permanently erases the account and associated data after re-authentication.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errRepoNotFound) {
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
	for _, key := range keys {
		if err := s.storage.Delete(key); err != nil {
			log.Printf(`{"event":"photo_purge_failed","key":%q,"error":%q}`, key, err.Error())
		}
	}
	s.terminator.Disconnect(userID)
	return nil
}

func (s *Service) issueSession(ctx context.Context, userID, userAgent, ip string) (Tokens, error) {
	refresh, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, userID, hashToken(refresh), userAgent, ip, time.Now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := signAccessToken(s.jwtSecret, userID, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

// ValidatePassword enforces the password policy: 8-128 chars with at least one letter and one digit.
func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
		return ErrWeakPassword
	}
	var letter, digit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !letter || !digit {
		return ErrWeakPassword
	}
	return nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidEmail checks a normalised address: a single plain addr-spec, at most 254 bytes.
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
		out[i] = resetCodeAlphabet[int(b)%len(resetCodeAlphabet)] // 256 % 32 == 0: no modulo bias
	}
	return string(out), nil
}

func hashToken(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
