package auth

import (
	"context"
	"errors"
	"net"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryEmailExists = errors.New("auth email already exists")
	ErrRepositoryNotFound    = errors.New("auth user not found")
	ErrRepositorySessionGone = errors.New("auth session not found")
	ErrRepositoryResetGone   = errors.New("auth reset code not found")
)

type StoredUser struct {
	ID        string
	Email     string
	Status    string
	CreatedAt time.Time
}

type StoredUserWithPassword struct {
	User         StoredUser
	PasswordHash string
}

type StoredSession struct {
	ID     string
	UserID string
}

type StoredReset struct {
	ID        string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	DeleteUser(ctx context.Context, userID string) error
	OnboardingState(ctx context.Context, userID string) (OnboardingState, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	RevokeUserSessions(ctx context.Context, userID string, at time.Time) error
	CheckSession(ctx context.Context, userID, sessionID string) error
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetLatestPasswordReset(ctx context.Context, userID string) (StoredReset, error)
	RegisterResetAttempt(ctx context.Context, resetID string) error
	ConsumePasswordReset(ctx context.Context, resetID, userID, newPasswordHash string, at time.Time) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return StoredUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var user StoredUser
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, status, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.Status, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	// Matching preferences always exist, with permissive defaults.
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in)
		VALUES ($1, ARRAY['man', 'woman', 'non_binary'])
	`, user.ID); err != nil {
		return StoredUser{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	var u StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, status, password_hash, created_at FROM users WHERE email = $1
	`, email).Scan(&u.User.ID, &u.User.Email, &u.User.Status, &u.PasswordHash, &u.User.CreatedAt)
	return u, mapNoRows(err)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	var u StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, status, password_hash, created_at FROM users WHERE id = $1
	`, userID).Scan(&u.User.ID, &u.User.Email, &u.User.Status, &u.PasswordHash, &u.User.CreatedAt)
	return u, mapNoRows(err)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	var u StoredUser
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, status, created_at FROM users WHERE id = $1
	`, userID).Scan(&u.ID, &u.Email, &u.Status, &u.CreatedAt)
	return u, mapNoRows(err)
}

// DeleteUser removes the user; every dependent table cascades.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) error {
	tag, err := r.dbPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) OnboardingState(ctx context.Context, userID string) (OnboardingState, error) {
	var s OnboardingState
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1),
		       (SELECT COUNT(*) FROM photos WHERE user_id = $1)
	`, userID).Scan(&s.HasProfile, &s.PhotoCount)
	s.Completed = s.HasProfile && s.PhotoCount > 0
	return s, err
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 300)), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	return s, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

// RotateSessionToken swaps the refresh token only if the caller still holds the
// current one, so two concurrent refreshes with the same token cannot both win.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = $7
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id
	`, sessionID, currentTokenHash, nextTokenHash, nullableString(truncate(userAgent, 300)), nullableIP(ipAddress), expiresAt, usedAt).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2, last_used_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash, revokedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) RevokeUserSessions(ctx context.Context, userID string, at time.Time) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL
	`, userID, at)
	return err
}

// CheckSession backs middleware.RequireUser: the session must be live and the account active.
func (r *PGRepository) CheckSession(ctx context.Context, userID, sessionID string) error {
	var status string
	err := r.dbPool.QueryRow(ctx, `
		SELECT u.status
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.user_id = $2 AND s.revoked_at IS NULL AND s.expires_at > NOW()
	`, sessionID, userID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return middleware.ErrSessionInactive
	}
	if err != nil {
		return err
	}
	if status != "active" {
		return middleware.ErrAccountSuspended
	}
	return nil
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Only the newest code is ever valid.
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetLatestPasswordReset(ctx context.Context, userID string) (StoredReset, error) {
	var s StoredReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, code_hash, attempts, expires_at FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&s.ID, &s.CodeHash, &s.Attempts, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredReset{}, ErrRepositoryResetGone
	}
	return s, err
}

func (r *PGRepository) RegisterResetAttempt(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

func (r *PGRepository) ConsumePasswordReset(ctx context.Context, resetID, userID, newPasswordHash string, at time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = $2 WHERE id = $1 AND used_at IS NULL`, resetID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryResetGone
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1`, userID, newPasswordHash, at); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`, userID, at); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func mapNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryNotFound
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func nullableIP(value string) any {
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.String()
}
