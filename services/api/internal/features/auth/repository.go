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

type StoredUser struct {
	ID           string
	Email        string
	IsDemo       bool
	PasswordHash string
	CreatedAt    time.Time
}

type StoredSession struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

type StoredResetCode struct {
	ID        string
	UserID    string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserByEmail(ctx context.Context, email string) (StoredUser, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	RevokeAllSessions(ctx context.Context, userID, exceptSessionID string) error
	CreateResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetActiveResetCode(ctx context.Context, userID string) (StoredResetCode, error)
	IncrementResetAttempts(ctx context.Context, codeID string) error
	ConsumeResetCode(ctx context.Context, codeID string) error
	DeleteUser(ctx context.Context, userID string) (photoKeys []string, err error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const userColumns = `id, email, is_demo, password_hash, created_at`

func scanUser(row pgx.Row) (StoredUser, error) {
	var u StoredUser
	err := row.Scan(&u.ID, &u.Email, &u.IsDemo, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUser{}, ErrRepositoryNotFound
	}
	return u, err
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return StoredUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := scanUser(tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING `+userColumns, email, passwordHash))
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO profiles (user_id) VALUES ($1)`, user.ID); err != nil {
		return StoredUser{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1)`, user.ID); err != nil {
		return StoredUser{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (StoredUser, error) {
	return scanUser(r.dbPool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	return scanUser(r.dbPool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, userID))
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	tag, err := r.dbPool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id, expires_at
	`, userID, tokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID, &s.ExpiresAt)
	return s, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id, expires_at
		FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&s.ID, &s.UserID, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING id, user_id, expires_at
	`, sessionID, nextTokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW(), last_used_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) RevokeAllSessions(ctx context.Context, userID, exceptSessionID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL AND ($2 = '' OR id::text <> $2)
	`, userID, exceptSessionID)
	return err
}

func (r *PGRepository) CreateResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Only the latest code is valid.
	if _, err := tx.Exec(ctx, `UPDATE password_reset_codes SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_reset_codes (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetActiveResetCode(ctx context.Context, userID string) (StoredResetCode, error) {
	var c StoredResetCode
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id, code_hash, attempts, expires_at
		FROM password_reset_codes
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&c.ID, &c.UserID, &c.CodeHash, &c.Attempts, &c.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredResetCode{}, ErrRepositoryNotFound
	}
	return c, err
}

func (r *PGRepository) IncrementResetAttempts(ctx context.Context, codeID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_reset_codes SET attempts = attempts + 1 WHERE id = $1`, codeID)
	return err
}

func (r *PGRepository) ConsumeResetCode(ctx context.Context, codeID string) error {
	tag, err := r.dbPool.Exec(ctx, `UPDATE password_reset_codes SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, codeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

// DeleteUser removes the account; foreign keys cascade to profile, photos, swipes,
// matches, conversations, messages, blocks and notifications. Reports are kept with
// the user reference nulled. The photo storage keys are returned so files can be erased.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) ([]string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrRepositoryNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return keys, nil
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

func nullableIP(value string) any {
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.String()
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
