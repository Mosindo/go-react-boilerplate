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
	ErrRepositoryResetBad    = errors.New("auth reset code invalid")
)

const maxResetAttempts = 5

type StoredUser struct {
	ID              string
	Email           string
	CreatedAt       time.Time
	ProfileComplete bool
}

type StoredUserWithPassword struct {
	User         StoredUser
	PasswordHash string
}

type StoredSession struct {
	ID     string
	UserID string
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	CreatePasswordReset(ctx context.Context, email, codeHash string, expiresAt time.Time) (bool, error)
	ConsumePasswordReset(ctx context.Context, email, codeHash, newPasswordHash string) error
	DeleteUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// profile_complete: a profile, its preferences and at least one photo exist.
const userColumns = `
	u.id, u.email, u.created_at,
	(EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = u.id)
	 AND EXISTS (SELECT 1 FROM preferences pr WHERE pr.user_id = u.id)
	 AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = u.id))`

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, created_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `u.email = $1`, email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `u.id = $1`, userID)
}

func (r *PGRepository) getUserAuth(ctx context.Context, where string, arg any) (StoredUserWithPassword, error) {
	var user StoredUserWithPassword
	err := r.dbPool.QueryRow(ctx, `SELECT `+userColumns+`, u.password_hash FROM users u WHERE `+where, arg).
		Scan(&user.User.ID, &user.User.Email, &user.User.CreatedAt, &user.User.ProfileComplete, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id = $1`, userID).
		Scan(&user.ID, &user.Email, &user.CreatedAt, &user.ProfileComplete)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUser{}, ErrRepositoryNotFound
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID)
	return session, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&session.ID, &session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
}

// RotateSessionToken swaps the refresh hash only if the presented one is still current, so a refresh
// token can be redeemed exactly once even under concurrent requests.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = $7
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id
	`, sessionID, currentTokenHash, nextTokenHash, nullableString(truncate(userAgent, 256)), nullableIP(ipAddress), expiresAt, usedAt).Scan(&session.ID, &session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositorySessionGone
		}
		return StoredSession{}, err
	}
	return session, nil
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

// UpdatePassword sets a new hash and revokes every session.
func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := setPassword(ctx, tx, userID, passwordHash); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func setPassword(ctx context.Context, tx pgx.Tx, userID, passwordHash string) error {
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

// CreatePasswordReset stores a fresh code for the email (invalidating older ones). It reports whether
// the account exists so the caller can decide to send mail, without exposing that to the client.
func (r *PGRepository) CreatePasswordReset(ctx context.Context, email, codeHash string, expiresAt time.Time) (bool, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`, userID, codeHash, expiresAt); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (r *PGRepository) ConsumePasswordReset(ctx context.Context, email, codeHash, newPasswordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var resetID, userID, storedHash string
	err = tx.QueryRow(ctx, `
		SELECT pr.id, pr.user_id, pr.code_hash
		FROM password_resets pr JOIN users u ON u.id = pr.user_id
		WHERE u.email = $1 AND pr.used_at IS NULL AND pr.expires_at > NOW() AND pr.attempts < $2
		ORDER BY pr.created_at DESC LIMIT 1
		FOR UPDATE OF pr
	`, email, maxResetAttempts).Scan(&resetID, &userID, &storedHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRepositoryResetBad
		}
		return err
	}

	if _, err := tx.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID); err != nil {
		return err
	}
	if !constantTimeEqual(storedHash, codeHash) {
		if err := tx.Commit(ctx); err != nil { // persist the failed attempt
			return err
		}
		return ErrRepositoryResetBad
	}
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE id = $1`, resetID); err != nil {
		return err
	}
	if err := setPassword(ctx, tx, userID, newPasswordHash); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteUser removes the account; every dependent row is removed by ON DELETE CASCADE.
// It returns the photo storage keys so the caller can delete the files.
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
	return keys, tx.Commit(ctx)
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

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
