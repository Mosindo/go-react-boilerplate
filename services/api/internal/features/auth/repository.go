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
	ID        string
	UserID    string
	ExpiresAt time.Time
}

type StoredResetCode struct {
	ID       string
	UserID   string
	CodeHash string
	Attempts int
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (User, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) ([]string, error)
	TouchActivity(ctx context.Context, userID string) error

	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) (string, error)
	RevokeUserSessions(ctx context.Context, userID, exceptSessionID string) error
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)

	CreateResetCode(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	CountRecentResetCodes(ctx context.Context, userID string, since time.Time) (int, error)
	GetActiveResetCode(ctx context.Context, userID string) (StoredResetCode, error)
	IncrementResetAttempts(ctx context.Context, codeID string) error
	ConsumeResetCode(ctx context.Context, codeID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// CreateUser inserts the account together with its empty profile and default
// preferences so every user always has both rows.
func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var user User
	err = tx.QueryRow(ctx, `
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
	if _, err := tx.Exec(ctx, `INSERT INTO profiles (user_id) VALUES ($1)`, user.ID); err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1)`, user.ID); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `WHERE email = $1`, email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `WHERE id = $1`, userID)
}

func (r *PGRepository) getUserAuth(ctx context.Context, where string, arg string) (StoredUserWithPassword, error) {
	var stored StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, is_demo, created_at, password_hash
		FROM users
		`+where, arg).Scan(&stored.User.ID, &stored.User.Email, &stored.User.IsDemo, &stored.User.CreatedAt, &stored.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return stored, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (User, error) {
	stored, err := r.GetUserAuthByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	return stored.User, nil
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

// DeleteUser removes the account. Foreign keys cascade to every personal row
// (profile, photos, swipes, matches, conversations, messages, notifications,
// sessions, blocks). Reports keep their row with the user reference nulled.
// The storage keys of deleted photos are returned so files can be removed.
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

func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id, expires_at
	`, userID, tokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID, &session.ExpiresAt)
	if err != nil {
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id, expires_at
		FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&session.ID, &session.UserID, &session.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

// RotateSessionToken swaps the refresh token atomically. Matching on the
// current hash makes concurrent reuse of the same refresh token fail.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = NOW()
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL
		RETURNING id, user_id, expires_at
	`, sessionID, currentTokenHash, nextTokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID, &session.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepositorySessionGone
		}
		return "", err
	}
	return userID, nil
}

func (r *PGRepository) RevokeUserSessions(ctx context.Context, userID, exceptSessionID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL AND ($2 = '' OR id::text <> $2)
	`, userID, exceptSessionID)
	return err
}

func (r *PGRepository) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id::text = $1 AND user_id::text = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&active)
	return active, err
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

func (r *PGRepository) CountRecentResetCodes(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	err := r.dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM password_reset_codes WHERE user_id = $1 AND created_at > $2
	`, userID, since).Scan(&count)
	return count, err
}

func (r *PGRepository) GetActiveResetCode(ctx context.Context, userID string) (StoredResetCode, error) {
	var code StoredResetCode
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id, code_hash, attempts
		FROM password_reset_codes
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&code.ID, &code.UserID, &code.CodeHash, &code.Attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredResetCode{}, ErrRepositoryNotFound
		}
		return StoredResetCode{}, err
	}
	return code, nil
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
	if len(value) > max {
		return value[:max]
	}
	return value
}
