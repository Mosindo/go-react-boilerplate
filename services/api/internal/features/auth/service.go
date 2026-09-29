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
	"net/url"
	"strings"
	"time"

	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrInvalidBirthDate    = errors.New("birthDate must be formatted YYYY-MM-DD")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetTokenTTL   = time.Hour
	maxEmailLen     = 254
)

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	baseURL   string
	dummyHash string
	now       func() time.Time
}

func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, appBaseURL string) *Service {
	dummy, _ := HashPassword("dummy-password-for-timing-equalisation")
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		mailer:    m,
		baseURL:   strings.TrimRight(appBaseURL, "/"),
		dummyHash: dummy,
		now:       time.Now,
	}
}

// NormalizeEmail trims and lower-cases an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	if email == "" || len(email) > maxEmailLen || strings.ContainsAny(email, " <>\r\n\t,;") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	return strings.Contains(email[strings.LastIndex(email, "@"):], ".")
}

func (s *Service) Register(ctx context.Context, email, password, birthDate, userAgent, ipAddress string) (Tokens, MeResponse, error) {
	email = NormalizeEmail(email)
	if !validEmail(email) {
		return Tokens{}, MeResponse{}, ErrInvalidEmail
	}
	if err := ValidatePassword(password); err != nil {
		return Tokens{}, MeResponse{}, err
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(birthDate))
	if err != nil {
		return Tokens{}, MeResponse{}, ErrInvalidBirthDate
	}
	if err := ValidateBirthDate(birth, s.now()); err != nil {
		return Tokens{}, MeResponse{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return Tokens{}, MeResponse{}, err
	}
	created, err := s.repo.CreateUser(ctx, email, hash, birth)
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, MeResponse{}, ErrEmailExists
		}
		return Tokens{}, MeResponse{}, err
	}
	return s.startSession(ctx, created.ID, userAgent, ipAddress)
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, MeResponse, error) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			// Equalise timing so the response does not reveal whether the email exists.
			CheckPassword(s.dummyHash, password)
			return Tokens{}, MeResponse{}, ErrInvalidCredentials
		}
		return Tokens{}, MeResponse{}, err
	}
	if !CheckPassword(stored.PasswordHash, password) {
		return Tokens{}, MeResponse{}, ErrInvalidCredentials
	}
	return s.startSession(ctx, stored.User.ID, userAgent, ipAddress)
}

func (s *Service) startSession(ctx context.Context, userID, userAgent, ipAddress string) (Tokens, MeResponse, error) {
	refreshToken, err := generateToken()
	if err != nil {
		return Tokens{}, MeResponse{}, err
	}
	session, err := s.repo.CreateSession(ctx, userID, hashToken(refreshToken), userAgent, ipAddress, s.now().Add(refreshTokenTTL))
	if err != nil {
		return Tokens{}, MeResponse{}, err
	}
	access, err := signAccessToken(s.jwtSecret, userID, session.ID, s.now())
	if err != nil {
		return Tokens{}, MeResponse{}, err
	}
	if err := s.repo.TouchUser(ctx, userID); err != nil {
		log.Printf(`{"event":"auth_touch_failed"}`)
	}
	me, err := s.repo.LoadMe(ctx, userID, s.now())
	if err != nil {
		return Tokens{}, MeResponse{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refreshToken}, me, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (Tokens, MeResponse, error) {
	current := hashToken(strings.TrimSpace(refreshToken))
	if current == "" {
		return Tokens{}, MeResponse{}, ErrInvalidRefreshToken
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, current)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, MeResponse{}, ErrInvalidRefreshToken
		}
		return Tokens{}, MeResponse{}, err
	}
	next, err := generateToken()
	if err != nil {
		return Tokens{}, MeResponse{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, current, hashToken(next), userAgent, ipAddress, s.now().Add(refreshTokenTTL))
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, MeResponse{}, ErrInvalidRefreshToken
		}
		return Tokens{}, MeResponse{}, err
	}
	access, err := signAccessToken(s.jwtSecret, rotated.UserID, rotated.ID, s.now())
	if err != nil {
		return Tokens{}, MeResponse{}, err
	}
	_ = s.repo.TouchUser(ctx, rotated.UserID)
	me, err := s.repo.LoadMe(ctx, rotated.UserID, s.now())
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return Tokens{}, MeResponse{}, ErrInvalidRefreshToken
		}
		return Tokens{}, MeResponse{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, me, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	h := hashToken(strings.TrimSpace(refreshToken))
	if h == "" {
		return ErrInvalidRefreshToken
	}
	if err := s.repo.RevokeSessionByTokenHash(ctx, h); err != nil && !errors.Is(err, ErrRepositorySessionGone) {
		return err
	}
	return nil
}

// Forgot issues a reset token and mails it. It never reveals whether the account exists:
// unknown emails, throttled requests and mail failures all end silently.
func (s *Service) Forgot(ctx context.Context, email string) {
	email = NormalizeEmail(email)
	if !validEmail(email) {
		return
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, ErrRepositoryNotFound) {
			log.Printf(`{"event":"auth_forgot_lookup_failed"}`)
		}
		return
	}
	token, err := generateToken()
	if err != nil {
		return
	}
	created, err := s.repo.CreateResetToken(ctx, stored.User.ID, hashToken(token), s.now().Add(resetTokenTTL))
	if err != nil {
		log.Printf(`{"event":"auth_forgot_store_failed"}`)
		return
	}
	if !created {
		return
	}
	body := "We received a request to reset your password.\n\n"
	if s.baseURL != "" {
		body += "Open this link within one hour:\n" + s.baseURL + "/reset-password?token=" + url.QueryEscape(token) + "\n\n"
	} else {
		body += "Your reset code (valid for one hour): " + token + "\n\n"
	}
	body += "If you did not ask for this, you can ignore this email."
	if err := s.mailer.Send(ctx, mailer.Message{To: stored.User.Email, Subject: "Reset your password", Body: body}); err != nil {
		log.Printf(`{"event":"auth_forgot_mail_failed"}`)
	}
}

// Reset sets a new password from a single-use token and revokes every session.
func (s *Service) Reset(ctx context.Context, token, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.repo.ConsumeResetToken(ctx, hashToken(strings.TrimSpace(token)), hash); err != nil {
		if errors.Is(err, ErrRepositoryResetToken) {
			return ErrInvalidResetToken
		}
		return err
	}
	return nil
}

func signAccessToken(secret []byte, userID, sessionID string, now time.Time) (string, error) {
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

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	if strings.TrimSpace(token) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
