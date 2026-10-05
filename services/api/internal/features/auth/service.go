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

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
	ErrUserNotFound        = errors.New("user not found")
	ErrWeakPassword        = errors.New("password must be between 8 and 72 bytes")
	ErrInvalidEmail        = errors.New("invalid email address")
)

const (
	accessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 30 * 24 * time.Hour
	resetTokenTTL    = time.Hour
	maxPasswordBytes = 72 // bcrypt silently ignores (or rejects) anything beyond 72 bytes
	minPasswordBytes = 8
)

// Mailer sends the password recovery email.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// AccountEraser lets other features clean up before an account is deleted (photo files, live sockets).
// Prepare runs before the user row is removed and returns a function executed after the removal succeeded.
type AccountEraser interface {
	Prepare(ctx context.Context, userID string) (finish func(), err error)
}

type Service struct {
	repo          Repository
	jwtSecret     []byte
	mailer        Mailer
	resetURL      string
	eraser        AccountEraser
	dummyHash     []byte
	asyncMailSend bool
	onRevoked     func(userID string)
}

type Option func(*Service)

// WithMailer enables password recovery emails. resetURL must contain "{token}".
func WithMailer(m Mailer, resetURL string) Option {
	return func(s *Service) { s.mailer, s.resetURL = m, resetURL }
}

// WithSessionsRevoked registers a callback run after every session of a user was revoked (password reset or
// change), so live WebSockets are closed instead of surviving until the access token expires.
func WithSessionsRevoked(fn func(userID string)) Option {
	return func(s *Service) { s.onRevoked = fn }
}

func WithAccountEraser(e AccountEraser) Option {
	return func(s *Service) { s.eraser = e }
}

// WithSyncMail sends recovery emails inline (tests); by default delivery is asynchronous so response
// time does not reveal whether an email address is registered.
func WithSyncMail() Option {
	return func(s *Service) { s.asyncMailSend = false }
}

func NewService(repo Repository, jwtSecret []byte, opts ...Option) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equalisation"), bcrypt.DefaultCost)
	s := &Service{repo: repo, jwtSecret: jwtSecret, dummyHash: dummy, asyncMailSend: true}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func validPassword(password string) bool {
	return len(password) >= minPasswordBytes && len(password) <= maxPasswordBytes
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ipAddress string) (Tokens, User, error) {
	email = normalizeEmail(email)
	if !validEmail(email) {
		return Tokens{}, User{}, ErrInvalidEmail
	}
	if !validPassword(password) {
		return Tokens{}, User{}, ErrWeakPassword
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
			// Burn the same CPU time as a real check so response time does not reveal registered emails.
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
	currentHash := hashToken(strings.TrimSpace(refreshToken))
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

	next, err := randomToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	rotated, err := s.repo.RotateSessionToken(ctx, session.ID, currentHash, hashToken(next), userAgent, ipAddress, time.Now().Add(refreshTokenTTL), time.Now())
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
	return Tokens{AccessToken: accessToken, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(strings.TrimSpace(refreshToken))
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

// IsSessionActive implements middleware.SessionChecker.
func (s *Service) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	return s.repo.IsSessionActive(ctx, sessionID, userID)
}

// ForgotPassword always succeeds from the caller's point of view: unknown addresses get no email and no error.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	if s.mailer == nil {
		return nil
	}
	stored, err := s.repo.GetUserAuthByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	token, err := randomToken()
	if err != nil {
		return err
	}
	created, err := s.repo.CreatePasswordReset(ctx, stored.User.ID, hashToken(token), time.Now().Add(resetTokenTTL))
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	link := strings.ReplaceAll(s.resetURL, "{token}", token)
	body := "Bonjour,\n\nPour choisir un nouveau mot de passe, ouvrez ce lien (valable 1 heure) :\n" + link +
		"\n\nSi vous n'êtes pas à l'origine de cette demande, ignorez ce message.\n"
	send := func() {
		if err := s.mailer.Send(context.Background(), stored.User.Email, "Réinitialisation de votre mot de passe", body); err != nil {
			log.Printf("auth: password reset email failed: %v", err)
		}
	}
	if s.asyncMailSend {
		go send()
	} else {
		send()
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if !validPassword(newPassword) {
		return ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	userID, err := s.repo.ConsumePasswordReset(ctx, hashToken(strings.TrimSpace(token)), string(hash))
	if err != nil {
		if errors.Is(err, ErrRepositoryTokenGone) {
			return ErrInvalidResetToken
		}
		return err
	}
	if s.onRevoked != nil {
		s.onRevoked(userID)
	}
	return nil
}

// ChangePassword verifies the current password and revokes every other session.
func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if !validPassword(next) {
		return ErrWeakPassword
	}
	stored, err := s.verifyPassword(ctx, userID, current)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, stored.User.ID, string(hash), sessionID); err != nil {
		return err
	}
	if s.onRevoked != nil {
		s.onRevoked(stored.User.ID) // the current device reconnects with its still-valid session
	}
	return nil
}

// DeleteAccount permanently removes the account after password confirmation.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	if _, err := s.verifyPassword(ctx, userID, password); err != nil {
		return err
	}
	finish := func() {}
	if s.eraser != nil {
		f, err := s.eraser.Prepare(ctx, userID)
		if err != nil {
			return err
		}
		finish = f
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	finish()
	return nil
}

func (s *Service) verifyPassword(ctx context.Context, userID, password string) (StoredUserWithPassword, error) {
	stored, err := s.repo.GetUserAuthByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return StoredUserWithPassword{}, ErrUserNotFound
		}
		return StoredUserWithPassword{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		return StoredUserWithPassword{}, ErrInvalidCredentials
	}
	return stored, nil
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
	accessToken, err := signAccessToken(s.jwtSecret, user, session.ID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
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

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	if strings.TrimSpace(token) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// normalizeEmail trims and lower-cases so "  Foo@Bar.com " and "foo@bar.com" are the same account.
func normalizeEmail(email string) string {
	// NUL bytes are invalid in PostgreSQL text and would turn a bad login into a 500.
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(email, "\x00", "")))
}

// validEmail accepts a plain "local@domain.tld" address (no display name, no comments, no spaces).
func validEmail(email string) bool {
	if len(email) > 254 || strings.ContainsAny(email, " \t\r\n<>,;\"") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	at := strings.LastIndex(email, "@")
	return at > 0 && strings.Contains(email[at+1:], ".") && !strings.HasSuffix(email, ".")
}

func toUser(stored StoredUser) User {
	return User{ID: stored.ID, Email: stored.Email, CreatedAt: stored.CreatedAt}
}
