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
	ErrRepositoryResetGone   = errors.New("auth reset token not found")
)

type StoredUser struct {
	ID               string
	Email            string
	Status           string
	ProfileCompleted bool
	CreatedAt        time.Time
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
	TouchUser(ctx context.Context, userID string) error
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
	RevokeUserSessions(ctx context.Context, userID, exceptSessionID string) error
	UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error
	CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string) (userID string, err error)
	DeleteUser(ctx context.Context, userID string) (storageKeys []string, err error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const userSelect = `
	SELECT u.id, u.email, u.status, u.created_at,
	       EXISTS (SELECT 1 FROM photos p WHERE p.user_id = u.id)
	FROM users u`

// authSelect additionally returns the password hash (last column).
const authSelect = `
	SELECT u.id, u.email, u.status, u.created_at,
	       EXISTS (SELECT 1 FROM photos p WHERE p.user_id = u.id),
	       u.password_hash
	FROM users u`

func scanUser(row pgx.Row, dest *StoredUser, extra ...any) error {
	targets := append([]any{&dest.ID, &dest.Email, &dest.Status, &dest.CreatedAt, &dest.ProfileCompleted}, extra...)
	return row.Scan(targets...)
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, status, created_at, FALSE
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.Status, &user.CreatedAt, &user.ProfileCompleted)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	var out StoredUserWithPassword
	err := scanUser(r.dbPool.QueryRow(ctx, authSelect+` WHERE u.email = $1`, email), &out.User, &out.PasswordHash)
	return out, mapNoRows(err)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	var out StoredUserWithPassword
	err := scanUser(r.dbPool.QueryRow(ctx, authSelect+` WHERE u.id = $1`, userID), &out.User, &out.PasswordHash)
	return out, mapNoRows(err)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	var out StoredUser
	err := scanUser(r.dbPool.QueryRow(ctx, userSelect+` WHERE u.id = $1`, userID), &out)
	return out, mapNoRows(err)
}

func (r *PGRepository) TouchUser(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, truncate(userAgent, 256), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	return s, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT s.id, s.user_id
		FROM sessions s
		JOIN users u ON u.id = s.user_id AND u.status = 'active'
		WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW()
	`, tokenHash).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

// RotateSessionToken swaps the refresh token only if the caller still holds the current one,
// so two concurrent refreshes with the same token cannot both succeed.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = NOW()
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL
		RETURNING id, user_id
	`, sessionID, currentTokenHash, nextTokenHash, truncate(userAgent, 256), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
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

func (r *PGRepository) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions s JOIN users u ON u.id = s.user_id
			WHERE s.id = $2 AND s.user_id = $1 AND s.revoked_at IS NULL
			  AND s.expires_at > NOW() AND u.status = 'active'
		)
	`, userID, sessionID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) RevokeUserSessions(ctx context.Context, userID, exceptSessionID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL AND ($2::uuid IS NULL OR id <> $2::uuid)
	`, userID, nullIfEmpty(exceptSessionID))
	return err
}

func (r *PGRepository) UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	return err
}

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ConsumePasswordReset atomically burns a valid token, sets the new password and revokes every session.
func (r *PGRepository) ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE password_resets SET used_at = NOW()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryResetGone
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, newPasswordHash); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
		return "", err
	}
	return userID, tx.Commit(ctx)
}

// DeleteUser removes the account; foreign keys cascade to profile, photos, swipes, matches,
// conversations, messages, blocks and notifications. Reports are kept anonymised (SET NULL).
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) ([]string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, `SELECT storage_key, thumb_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	var keys []string
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, a, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return nil, err
	}
	return keys, tx.Commit(ctx)
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

func truncate(value string, max int) any {
	if value == "" {
		return nil
	}
	if len(value) > max {
		return value[:max]
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

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
