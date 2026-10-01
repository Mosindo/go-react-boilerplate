package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"example.com/api/internal/platform/mailer"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrAccountSuspended    = errors.New("account suspended")
	ErrInvalidResetCode    = errors.New("invalid or expired reset code")
	ErrInvalidEmail        = errors.New("invalid email address")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

const resetCodeTTL = 15 * time.Minute

// resetAlphabet avoids look-alike characters (0/O, 1/I/L) so codes survive typing.
const resetAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	appName   string
}

func NewService(repo Repository, jwtSecret []byte) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// SetMailer enables password recovery emails.
func (s *Service) SetMailer(m mailer.Mailer, appName string) {
	s.mailer = m
	s.appName = appName
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	normalizedEmail := normalizeEmail(email)
	if !validEmail(normalizedEmail) {
		return Tokens{}, User{}, ErrInvalidEmail
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
			return Tokens{}, User{}, ErrInvalidCredentials
		}
		return Tokens{}, User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, User{}, ErrInvalidCredentials
	}

	if stored.User.Status != "active" {
		return Tokens{}, User{}, ErrAccountSuspended
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

	if userRecord.Status != "active" {
		return Tokens{}, User{}, ErrAccountSuspended
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

// validEmail accepts a bare address with a dotted domain (no display names).
func validEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return false
	}
	at := strings.LastIndexByte(email, '@')
	return at > 0 && strings.Contains(email[at+1:], ".") && !strings.HasSuffix(email, ".")
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func toUser(stored StoredUser) User {
	return User{
		ID:             stored.ID,
		Email:          stored.Email,
		OrganizationID: stored.OrganizationID,
		Role:           stored.Role,
		HasProfile:     stored.HasProfile,
		PhotoCount:     stored.PhotoCount,
		CreatedAt:      stored.CreatedAt,
	}
}

// ForgotPassword emails a short-lived recovery code. It never reveals whether
// the address is registered and never returns an error to the caller.
func (s *Service) ForgotPassword(ctx context.Context, email string) {
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil || stored.User.Status != "active" || s.mailer == nil {
		return
	}
	code, err := generateResetCode()
	if err != nil {
		return
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashRefreshToken(code), time.Now().Add(resetCodeTTL)); err != nil {
		return
	}
	body := "Votre code de récupération " + s.appName + " : " + code +
		"\n\nIl est valable 15 minutes. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message."
	to := stored.User.Email
	// Sent asynchronously so response time does not betray whether the account exists.
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = s.mailer.Send(sendCtx, to, "Récupération de votre compte "+s.appName, body+"\n")
	}()
}

func (s *Service) ResetPassword(ctx context.Context, code, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = s.repo.ResetPassword(ctx, hashRefreshToken(strings.ToUpper(strings.TrimSpace(code))), string(hash))
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrInvalidResetCode
	}
	return err
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	stored, err := s.repo.GetPasswordHash(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(current)); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash), sessionID)
}

func generateResetCode() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = resetAlphabet[int(b)%len(resetAlphabet)]
	}
	return string(out), nil
}
