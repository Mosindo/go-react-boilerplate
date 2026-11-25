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
	errRepoEmailExists = errors.New("email exists")
	errRepoNotFound    = errors.New("not found")
)

type storedUser struct {
	User
	PasswordHash string
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (storedUser, error)
	GetUserByID(ctx context.Context, userID string) (storedUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ip string, expiresAt time.Time) (string, error)
	RotateSession(ctx context.Context, oldHash, newHash, userAgent, ip string, expiresAt time.Time) (sessionID, userID string, err error)
	RevokeSessionByHash(ctx context.Context, tokenHash string) error
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
	UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error
	CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ConsumeResetToken(ctx context.Context, tokenHash, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) (photoKeys []string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, is_demo, created_at
	`, email, passwordHash).Scan(&u.ID, &u.Email, &u.IsDemo, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, errRepoEmailExists
		}
		return User{}, err
	}
	return u, nil
}

func (r *PGRepository) getUser(ctx context.Context, where string, arg any) (storedUser, error) {
	var u storedUser
	err := r.db.QueryRow(ctx, `
		SELECT id, email, is_demo, created_at, password_hash FROM users WHERE `+where, arg,
	).Scan(&u.ID, &u.Email, &u.IsDemo, &u.CreatedAt, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedUser{}, errRepoNotFound
	}
	return u, err
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (storedUser, error) {
	return r.getUser(ctx, "email = $1", email)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (storedUser, error) {
	return r.getUser(ctx, "id = $1", userID)
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ip string, expiresAt time.Time) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`, userID, tokenHash, nullable(truncate(userAgent, 256)), nullableIP(ip), expiresAt).Scan(&id)
	if err != nil {
		return "", err
	}
	_, _ = r.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)
	return id, nil
}

// RotateSession atomically swaps the refresh token hash of an active session.
// A reused (already rotated) token matches no row and is rejected.
func (r *PGRepository) RotateSession(ctx context.Context, oldHash, newHash, userAgent, ip string, expiresAt time.Time) (string, string, error) {
	var sessionID, userID string
	err := r.db.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id
	`, oldHash, newHash, nullable(truncate(userAgent, 256)), nullableIP(ip), expiresAt).Scan(&sessionID, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", errRepoNotFound
	}
	if err != nil {
		return "", "", err
	}
	_, _ = r.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)
	return sessionID, userID, nil
}

func (r *PGRepository) RevokeSessionByHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	return err
}

func (r *PGRepository) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	var active bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&active)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" { // malformed uuid
			return false, nil
		}
		return false, err
	}
	return active, nil
}

// UpdatePassword changes the hash and revokes every session except keepSessionID.
func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE sessions SET revoked_at = NOW()
			WHERE user_id = $1 AND revoked_at IS NULL AND id::text <> $2
		`, userID, keepSessionID)
		return err
	})
}

func (r *PGRepository) CreateResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Only the latest code is valid.
		if _, err := tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt)
		return err
	})
}

// ConsumeResetToken sets a new password if the code is valid, marks the code
// used and revokes every session of the account.
func (r *PGRepository) ConsumeResetToken(ctx context.Context, tokenHash, passwordHash string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var userID string
		err := tx.QueryRow(ctx, `
			UPDATE password_reset_tokens SET used_at = NOW()
			WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
			RETURNING user_id
		`, tokenHash).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return errRepoNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
		return err
	})
}

// DeleteUser removes the account; every owned row is removed by ON DELETE
// CASCADE. Photo storage keys are returned so files can be erased afterwards.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) ([]string, error) {
	var keys []string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
		if err != nil {
			return err
		}
		keys, err = pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errRepoNotFound
		}
		return nil
	})
	return keys, err
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nullableIP(v string) any {
	ip := net.ParseIP(v)
	if ip == nil {
		return nil
	}
	return ip.String()
}

func truncate(v string, max int) string {
	if len(v) > max {
		return v[:max]
	}
	return v
}
