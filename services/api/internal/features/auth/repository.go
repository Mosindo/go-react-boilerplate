package auth

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryEmailExists = errors.New("auth email already exists")
	ErrRepositoryNotFound    = errors.New("auth user not found")
	ErrRepositorySessionGone = errors.New("auth session not found")
)

type StoredUserWithPassword struct {
	User         User
	PasswordHash string
}

type StoredSession struct {
	ID     string
	UserID string
}

type StoredReset struct {
	ID       string
	CodeHash string
	Attempts int
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (User, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) error
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	CreateReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetActiveReset(ctx context.Context, userID string) (StoredReset, error)
	RegisterResetAttempt(ctx context.Context, resetID string) error
	CompleteReset(ctx context.Context, resetID, userID, passwordHash string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var user User
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrRepositoryEmailExists
		}
		return User{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	var u StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at FROM users WHERE email = $1
	`, email).Scan(&u.User.ID, &u.User.Email, &u.PasswordHash, &u.User.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return u, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (User, error) {
	var u User
	err := r.dbPool.QueryRow(ctx, `SELECT id, email, created_at FROM users WHERE id = $1`, userID).
		Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrRepositoryNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		WITH touched AS (UPDATE users SET last_active_at = NOW() WHERE id = $1 RETURNING id)
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		SELECT id, $2, $3, $4, $5, NOW() FROM touched
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositoryNotFound
		}
		return StoredSession{}, err
	}
	return s, nil
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&s.ID, &s.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return s, nil
}

func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE sessions
		SET token_hash = $2, user_agent = COALESCE($3, user_agent), ip_address = COALESCE($4, ip_address),
		    expires_at = $5, last_used_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`, sessionID, nextTokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&active)
	return active, err
}

func (r *PGRepository) CreateReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetActiveReset(ctx context.Context, userID string) (StoredReset, error) {
	var s StoredReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, code_hash, attempts FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&s.ID, &s.CodeHash, &s.Attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredReset{}, ErrRepositoryNotFound
		}
		return StoredReset{}, err
	}
	return s, nil
}

func (r *PGRepository) RegisterResetAttempt(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

// CompleteReset sets the new password and revokes every session of the user.
func (r *PGRepository) CompleteReset(ctx context.Context, resetID, userID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE id = $1`, resetID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	if len(s) > n {
		return s[:n]
	}
	return s
}

func nullableIP(value string) any {
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	return nil
}
