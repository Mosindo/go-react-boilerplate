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

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/jwtauth"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/validate"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

var errBadResetCode = httpx.BadRequest("invalid or expired code")

type Service struct {
	repo      Repository
	jwtSecret []byte
	mailer    mailer.Mailer
	cost      int
	dummyHash []byte
	syncMail  bool
}

// NewService builds the auth service. cost is the bcrypt cost (0 = default).
func NewService(repo Repository, jwtSecret []byte, m mailer.Mailer, cost int) *Service {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	if m == nil {
		m = mailer.Log{}
	}
	s := &Service{repo: repo, jwtSecret: jwtSecret, mailer: m, cost: cost}
	// A hash of a random value at the same cost, compared against when the
	// account does not exist so login timing does not reveal registered emails.
	random := make([]byte, 16)
	_, _ = rand.Read(random)
	s.dummyHash, _ = bcrypt.GenerateFromPassword(prehash(hex.EncodeToString(random)), cost)
	return s
}

// SetSynchronousMail makes reset emails go out before Forgot returns (tests).
func (s *Service) SetSynchronousMail(v bool) { s.syncMail = v }

// prehash removes bcrypt's 72 byte input limit: passwords may be up to 128 chars.
func prehash(pw string) []byte {
	sum := sha256.Sum256([]byte(pw))
	return []byte(base64.StdEncoding.EncodeToString(sum[:]))
}

func (s *Service) hash(pw string) (string, error) { return HashPassword(pw, s.cost) }

// HashPassword is the single way passwords are hashed (bcrypt over a SHA-256
// pre-hash). Exported for tooling such as cmd/seed so hashes stay compatible.
func HashPassword(pw string, cost int) (string, error) {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	h, err := bcrypt.GenerateFromPassword(prehash(pw), cost)
	return string(h), err
}

func checkPassword(hash []byte, pw string) bool {
	return bcrypt.CompareHashAndPassword(hash, prehash(pw)) == nil
}

func (s *Service) Register(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	normalized, err := validate.Email(email)
	if err != nil {
		return Tokens{}, User{}, httpx.BadRequest(err.Error())
	}
	if err := validate.Password(password); err != nil {
		return Tokens{}, User{}, httpx.BadRequest(err.Error())
	}
	hash, err := s.hash(password)
	if err != nil {
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.CreateUser(ctx, normalized, hash)
	if err != nil {
		if errors.Is(err, ErrRepositoryEmailExists) {
			return Tokens{}, User{}, httpx.Conflict("email already registered")
		}
		return Tokens{}, User{}, err
	}
	user := toUser(stored)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (Tokens, User, error) {
	invalid := httpx.Unauthorized("invalid credentials")
	normalized := validate.NormalizeEmail(email)
	if len(normalized) > validate.MaxEmailLen || len(password) > 4*validate.MaxPasswordLen {
		return Tokens{}, User{}, invalid
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			checkPassword(s.dummyHash, password) // burn the same CPU as a real check
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}
	if !checkPassword([]byte(stored.PasswordHash), password) {
		return Tokens{}, User{}, invalid
	}
	user := toUser(stored)
	tokens, err := s.issueSessionTokens(ctx, user, userAgent, ip)
	return tokens, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (Tokens, User, error) {
	invalid := httpx.Unauthorized("invalid refresh token")
	current := hashRefreshToken(refreshToken)
	if current == "" {
		return Tokens{}, User{}, invalid
	}
	session, err := s.repo.GetActiveSessionByTokenHash(ctx, current)
	if err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}
	stored, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}
	next, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, User{}, err
	}
	if err := s.repo.RotateSessionToken(ctx, session.ID, current, hashRefreshToken(next), userAgent, ip, time.Now().Add(RefreshTokenTTL)); err != nil {
		if errors.Is(err, ErrRepositorySessionGone) {
			return Tokens{}, User{}, invalid
		}
		return Tokens{}, User{}, err
	}
	user := toUser(stored)
	access, err := jwtauth.Sign(s.jwtSecret, user.ID, session.ID, AccessTokenTTL, time.Now())
	if err != nil {
		return Tokens{}, User{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: next}, user, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	h := hashRefreshToken(refreshToken)
	if h == "" {
		return nil
	}
	return s.repo.RevokeSessionByTokenHash(ctx, h)
}

// ForgotPassword always succeeds from the caller's point of view. The email is
// sent in the background so response time does not reveal whether the account exists.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	normalized := validate.NormalizeEmail(email)
	if normalized == "" || len(normalized) > validate.MaxEmailLen {
		return nil
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil
		}
		return err
	}
	code, err := GenerateResetCode()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, stored.ID, HashResetCode(s.jwtSecret, stored.ID, code), time.Now().Add(ResetCodeTTL)); err != nil {
		return err
	}
	msg := mailer.Message{
		To:      stored.Email,
		Subject: "Your password reset code",
		Body: "Your password reset code is " + code + ".\n" +
			"It expires in 30 minutes and can be used once.\n" +
			"If you did not ask for this, you can ignore this email.",
	}
	send := func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.mailer.Send(sendCtx, msg); err != nil {
			log.Printf(`{"event":"reset_mail_failed","error":%q}`, err.Error())
		}
	}
	if s.syncMail {
		send()
	} else {
		go send()
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if err := validate.Password(newPassword); err != nil {
		return httpx.BadRequest(err.Error())
	}
	normalized := validate.NormalizeEmail(email)
	code = NormalizeResetCode(code)
	if normalized == "" || len(code) != ResetCodeLength {
		return errBadResetCode
	}
	stored, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return errBadResetCode
		}
		return err
	}
	reset, err := s.repo.ClaimResetAttempt(ctx, stored.ID, ResetMaxAttempts)
	if err != nil {
		if errors.Is(err, ErrRepositoryResetGone) {
			return errBadResetCode
		}
		return err
	}
	if !ResetCodeMatches(s.jwtSecret, stored.ID, code, reset.CodeHash) {
		return errBadResetCode
	}
	hash, err := s.hash(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.CompleteReset(ctx, reset.ID, stored.ID, hash); err != nil {
		if errors.Is(err, ErrRepositoryResetGone) {
			return errBadResetCode
		}
		return err
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := validate.Password(next); err != nil {
		return httpx.BadRequest(err.Error())
	}
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return httpx.Unauthorized("invalid token")
		}
		return err
	}
	if len(current) > 4*validate.MaxPasswordLen || !checkPassword([]byte(stored.PasswordHash), current) {
		return httpx.Forbidden("current password is incorrect")
	}
	hash, err := s.hash(next)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordRevokeOthers(ctx, userID, hash, sessionID)
}

func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
	stored, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return httpx.Unauthorized("invalid token")
		}
		return err
	}
	if len(password) > 4*validate.MaxPasswordLen || !checkPassword([]byte(stored.PasswordHash), password) {
		return httpx.Forbidden("password is incorrect")
	}
	return s.repo.DeleteUser(ctx, userID)
}

func (s *Service) issueSessionTokens(ctx context.Context, user User, userAgent, ip string) (Tokens, error) {
	refresh, err := generateRefreshToken()
	if err != nil {
		return Tokens{}, err
	}
	session, err := s.repo.CreateSession(ctx, user.ID, hashRefreshToken(refresh), userAgent, ip, time.Now().Add(RefreshTokenTTL))
	if err != nil {
		return Tokens{}, err
	}
	access, err := jwtauth.Sign(s.jwtSecret, user.ID, session.ID, AccessTokenTTL, time.Now())
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashRefreshToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 512 {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func toUser(s StoredUser) User {
	return User{ID: s.ID, Email: s.Email, CreatedAt: s.CreatedAt.UTC()}
}
