package auth

import (
	"context"
	"errors"
	"net"
	"time"

	perr "example.com/api/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryEmailExists  = errors.New("auth email already exists")
	ErrRepositoryNotFound     = errors.New("auth user not found")
	ErrRepositorySessionGone  = errors.New("auth session not found")
	ErrRepositoryResetInvalid = errors.New("auth reset code invalid")
)

type StoredUserWithPassword struct {
	User         User
	PasswordHash string
	Status       string
}

type StoredSession struct {
	ID     string
	UserID string
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string, birthDate time.Time) (User, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (User, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string, now time.Time) error
	UpdatePasswordRevokeOthers(ctx context.Context, userID, passwordHash, keepSessionID string, now time.Time) error
	DeleteUser(ctx context.Context, userID string) (mediaKeys []string, err error)
	SessionActive(ctx context.Context, sessionID string) (bool, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const userColumns = `u.id, u.email, u.birth_date, u.created_at,
	EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = u.id),
	(SELECT COUNT(*) FROM photos ph WHERE ph.user_id = u.id)`

func scanUser(row pgx.Row, dst *User) error {
	return row.Scan(&dst.ID, &dst.Email, &dst.BirthDate, &dst.CreatedAt, &dst.HasProfile, &dst.PhotoCount)
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string, birthDate time.Time) (User, error) {
	var user User
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, birth_date)
		VALUES ($1, $2, $3)
		RETURNING id, email, birth_date, created_at, FALSE, 0
	`, email, passwordHash, birthDate).Scan(&user.ID, &user.Email, &user.BirthDate, &user.CreatedAt, &user.HasProfile, &user.PhotoCount)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrRepositoryEmailExists
		}
		return User{}, err
	}
	return user, nil
}

func (r *PGRepository) GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `WHERE u.email = $1`, email)
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	return r.getUserAuth(ctx, `WHERE u.id = $1`, userID)
}

func (r *PGRepository) getUserAuth(ctx context.Context, where string, arg any) (StoredUserWithPassword, error) {
	var out StoredUserWithPassword
	row := r.dbPool.QueryRow(ctx, `SELECT `+userColumns+`, u.password_hash, u.status FROM users u `+where, arg)
	err := row.Scan(&out.User.ID, &out.User.Email, &out.User.BirthDate, &out.User.CreatedAt,
		&out.User.HasProfile, &out.User.PhotoCount, &out.PasswordHash, &out.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return out, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (User, error) {
	var user User
	err := scanUser(r.dbPool.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id = $1`, userID), &user)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrRepositoryNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		WITH touched AS (
			UPDATE users SET last_active_at = NOW() WHERE id = $1 RETURNING id
		)
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		SELECT id, $2, $3, $4, $5, NOW() FROM touched
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositoryNotFound
	}
	return session, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&session.ID, &session.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return session, err
}

func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = $6
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING id, user_id
	`, sessionID, nextTokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt, usedAt).Scan(&session.ID, &session.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	if err == nil {
		_, _ = r.dbPool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, session.UserID)
	}
	return session, err
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

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `DELETE FROM password_resets WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) ConsumePasswordReset(ctx context.Context, tokenHash, newPasswordHash string, now time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE password_resets SET used_at = $2
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
		RETURNING user_id
	`, tokenHash, now).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryResetInvalid
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, newPasswordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`, userID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdatePasswordRevokeOthers(ctx context.Context, userID, passwordHash, keepSessionID string, now time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = $3 WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL`, userID, keepSessionID, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

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

	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, perr.NotFound("user_not_found", "account not found")
	}
	return keys, tx.Commit(ctx)
}

// SessionActive is the revocation check behind every authenticated request.
func (r *PGRepository) SessionActive(ctx context.Context, sessionID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions s JOIN users u ON u.id = s.user_id
			WHERE s.id = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW() AND u.status = 'active'
		)`, sessionID).Scan(&ok)
	return ok, err
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
