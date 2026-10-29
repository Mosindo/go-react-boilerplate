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

type StoredReset struct {
	ID        string
	UserID    string
	CodeHash  string
	Attempts  int
	CreatedAt time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (User, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	SessionActive(ctx context.Context, sessionID, userID string) (bool, error)
	RevokeAllSessions(ctx context.Context, userID string) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	GetLatestPasswordReset(ctx context.Context, userID string) (StoredReset, error)
	RegisterResetAttempt(ctx context.Context, resetID string) error
	ConsumePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) (photoKeys []string, err error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const userSelect = `
	SELECT u.id, u.email, u.created_at,
	       EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = u.id),
	       EXISTS (SELECT 1 FROM preferences pr WHERE pr.user_id = u.id),
	       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = u.id)
	FROM users u`

func scanUser(row pgx.Row, extra ...any) (User, error) {
	var u User
	dest := append([]any{&u.ID, &u.Email, &u.CreatedAt, &u.HasProfile, &u.HasPreferences, &u.HasPhotos}, extra...)
	err := row.Scan(dest...)
	return u, err
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	user := User{Email: email}
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, created_at
	`, email, passwordHash).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrRepositoryEmailExists
		}
		return User{}, err
	}
	return user, nil
}

func (r *PGRepository) getAuth(ctx context.Context, where string, arg any) (StoredUserWithPassword, error) {
	var hash string
	user, err := scanUser(r.dbPool.QueryRow(ctx, `
		SELECT u.id, u.email, u.created_at,
		       EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = u.id),
		       EXISTS (SELECT 1 FROM preferences pr WHERE pr.user_id = u.id),
		       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = u.id),
		       u.password_hash
		FROM users u WHERE `+where, arg), &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return StoredUserWithPassword{User: user, PasswordHash: hash}, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	return r.getAuth(ctx, "u.email = $1", email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getAuth(ctx, "u.id = $1", userID)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (User, error) {
	user, err := scanUser(r.dbPool.QueryRow(ctx, userSelect+" WHERE u.id = $1", userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrRepositoryNotFound
		}
		return User{}, err
	}
	return user, nil
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
		SELECT id, user_id, expires_at FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&s.ID, &s.UserID, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = $6
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING id, user_id, expires_at
	`, sessionID, nextTokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt, usedAt).Scan(&s.ID, &s.UserID, &s.ExpiresAt)
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

// SessionActive checks the session and, at most every five minutes, refreshes the
// user's last-activity timestamp (used to rank recently active profiles).
func (r *PGRepository) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		WITH s AS (
			SELECT id FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		), touch AS (
			UPDATE profiles SET last_active_at = NOW()
			WHERE user_id = $2 AND last_active_at < NOW() - INTERVAL '5 minutes'
			  AND EXISTS (SELECT 1 FROM s)
			RETURNING 1
		)
		SELECT EXISTS (SELECT 1 FROM s)
	`, sessionID, userID).Scan(&active)
	return active, err
}

func (r *PGRepository) RevokeAllSessions(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	return err
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetLatestPasswordReset(ctx context.Context, userID string) (StoredReset, error) {
	var s StoredReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id, code_hash, attempts, created_at FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&s.ID, &s.UserID, &s.CodeHash, &s.Attempts, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredReset{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) RegisterResetAttempt(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

// ConsumePasswordReset atomically marks the code used, sets the new password and
// revokes every session of the account.
func (r *PGRepository) ConsumePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	tag, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, resetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteUser removes the account; foreign keys cascade to every owned row
// (profile, photos, swipes, matches, messages, blocks, reports, notifications).
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) ([]string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

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
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
