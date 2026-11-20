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
	ErrRepositoryNotFound    = errors.New("auth record not found")
)

type StoredUser struct {
	User
	PasswordHash string
}

type StoredSession struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

type StoredReset struct {
	ID       string
	CodeHash string
	Attempts int
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (StoredUser, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	RotateSession(ctx context.Context, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	LatestResetCreatedAt(ctx context.Context, userID string) (*time.Time, error)
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetActivePasswordReset(ctx context.Context, userID string, maxAttempts int) (StoredReset, error)
	IncrementResetAttempts(ctx context.Context, resetID string) error
	CompletePasswordReset(ctx context.Context, userID, resetID, passwordHash string) error
	UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error
	DeleteUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const userColumns = `
	u.id, u.email, u.password_hash, u.is_demo, u.created_at,
	(EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = u.id)
	 AND EXISTS (SELECT 1 FROM preferences pr WHERE pr.user_id = u.id)
	 AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = u.id)) AS profile_complete`

func scanUser(row pgx.Row) (StoredUser, error) {
	var u StoredUser
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsDemo, &u.CreatedAt, &u.ProfileComplete)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUser{}, ErrRepositoryNotFound
	}
	return u, err
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var user User
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, is_demo, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.IsDemo, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrRepositoryEmailExists
		}
		return User{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (StoredUser, error) {
	return scanUser(r.dbPool.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM users u
		WHERE u.email = $1 AND u.moderation_status <> 'suspended'
	`, email))
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	return scanUser(r.dbPool.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM users u
		WHERE u.id = $1 AND u.moderation_status <> 'suspended'
	`, userID))
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		WITH touched AS (
			UPDATE users SET last_active_at = NOW() WHERE id = $1 RETURNING id
		)
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		SELECT id, $2, $3, $4, $5, NOW() FROM touched
		RETURNING id, user_id, expires_at
	`, userID, tokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID, &session.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositoryNotFound
	}
	return session, err
}

// RotateSession atomically swaps the refresh token hash. Presenting an old
// (already rotated) token finds no row, so a refresh token works only once.
func (r *PGRepository) RotateSession(ctx context.Context, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions s
		SET token_hash = $2,
		    user_agent = COALESCE($3, s.user_agent),
		    ip_address = COALESCE($4, s.ip_address),
		    expires_at = $5,
		    last_used_at = NOW()
		FROM users u
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
		  AND u.id = s.user_id
		  AND u.moderation_status <> 'suspended'
		RETURNING s.id, s.user_id, s.expires_at
	`, currentTokenHash, nextTokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID, &session.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositoryNotFound
	}
	if err == nil {
		_, _ = r.dbPool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, session.UserID)
	}
	return session, err
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

// IsSessionActive is called on every authenticated request. It also refreshes
// users.last_active_at at most every five minutes (used for ranking).
func (r *PGRepository) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		WITH s AS (
			SELECT 1
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.id = $1 AND s.user_id = $2
			  AND s.revoked_at IS NULL AND s.expires_at > NOW()
			  AND u.moderation_status <> 'suspended'
		), touched AS (
			UPDATE users SET last_active_at = NOW()
			WHERE id = $2 AND EXISTS (SELECT 1 FROM s)
			  AND last_active_at < NOW() - INTERVAL '5 minutes'
			RETURNING 1
		)
		SELECT EXISTS (SELECT 1 FROM s)
	`, sessionID, userID).Scan(&active)
	if err != nil {
		var pgErr *pgconn.PgError
		// A malformed UUID can only come from a forged token: treat as inactive.
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return false, nil
		}
		return false, err
	}
	return active, nil
}

func (r *PGRepository) LatestResetCreatedAt(ctx context.Context, userID string) (*time.Time, error) {
	var createdAt *time.Time
	err := r.dbPool.QueryRow(ctx, `
		SELECT MAX(created_at) FROM password_reset_tokens WHERE user_id = $1
	`, userID).Scan(&createdAt)
	return createdAt, err
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Only the most recent code is valid.
	if _, err := tx.Exec(ctx, `
		UPDATE password_reset_tokens SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
	`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetActivePasswordReset(ctx context.Context, userID string, maxAttempts int) (StoredReset, error) {
	var reset StoredReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, code_hash, attempts
		FROM password_reset_tokens
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW() AND attempts < $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, maxAttempts).Scan(&reset.ID, &reset.CodeHash, &reset.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredReset{}, ErrRepositoryNotFound
	}
	return reset, err
}

func (r *PGRepository) IncrementResetAttempts(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_reset_tokens SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

func (r *PGRepository) CompletePasswordReset(ctx context.Context, userID, resetID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE password_reset_tokens SET used_at = NOW()
		WHERE id = $1 AND user_id = $2 AND used_at IS NULL
	`, resetID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	// A password reset signs the account out everywhere.
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL
	`, userID, keepSessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteUser removes the account; foreign keys cascade to every related row
// (profile, photos metadata, swipes, matches, messages, blocks, reports...).
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	if len(value) > 512 {
		return value[:512]
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
