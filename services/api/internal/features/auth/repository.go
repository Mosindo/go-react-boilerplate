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
	ID           string
	Email        string
	PasswordHash string
	HasProfile   bool
	CreatedAt    time.Time
}

type StoredSession struct {
	ID     string
	UserID string
}

type StoredReset struct {
	ID       string
	UserID   string
	CodeHash string
	Attempts int
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserByEmail(ctx context.Context, email string) (StoredUser, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	DeleteUser(ctx context.Context, userID string) (photoKeys []string, err error)

	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentHash, nextHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	RevokeAllSessions(ctx context.Context, userID string) error
	IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error)

	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	CountRecentPasswordResets(ctx context.Context, userID string, since time.Time) (int, error)
	ReserveResetAttempt(ctx context.Context, userID string, maxAttempts int) (StoredReset, error)
	ConsumePasswordReset(ctx context.Context, resetID string) (bool, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const userColumns = `u.id, u.email, u.password_hash, EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = u.id), u.created_at`

func scanUser(row pgx.Row) (StoredUser, error) {
	var u StoredUser
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.HasProfile, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUser{}, ErrRepositoryNotFound
	}
	return u, err
}

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	var u StoredUser
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at
	`, email, passwordHash).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	return u, nil
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (StoredUser, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.email = $1`, email))
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id = $1`, userID))
}

func (r *PGRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

// DeleteUser removes the account; every dependent row is removed by ON DELETE
// CASCADE (profile, photos, swipes, matches, messages, blocks, notifications).
// The photo storage keys are returned so the caller can purge the files.
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
			return ErrRepositoryNotFound
		}
		return nil
	})
	return keys, err
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	return s, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositoryNotFound
	}
	return s, err
}

// RotateSessionToken swaps the refresh token hash atomically. Matching on the
// current hash makes concurrent refreshes with the same token fail safely.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentHash, nextHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.db.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = NOW()
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id
	`, sessionID, currentHash, nextHash, nullableString(truncate(userAgent, 255)), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`, tokenHash)
	return err
}

func (r *PGRepository) RevokeAllSessions(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func (r *PGRepository) IsSessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
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

func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Only the latest code is valid.
		if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`, userID, codeHash, expiresAt)
		return err
	})
}

func (r *PGRepository) CountRecentPasswordResets(ctx context.Context, userID string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM password_resets WHERE user_id = $1 AND created_at > $2`, userID, since).Scan(&n)
	return n, err
}

// ReserveResetAttempt atomically consumes one attempt on the latest active
// reset code and returns it; ErrRepositoryNotFound when none is usable.
func (r *PGRepository) ReserveResetAttempt(ctx context.Context, userID string, maxAttempts int) (StoredReset, error) {
	var s StoredReset
	err := r.db.QueryRow(ctx, `
		UPDATE password_resets SET attempts = attempts + 1
		WHERE id = (
			SELECT id FROM password_resets
			WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
			ORDER BY created_at DESC LIMIT 1
		)
		  AND attempts < $2 AND used_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id, code_hash, attempts
	`, userID, maxAttempts).Scan(&s.ID, &s.UserID, &s.CodeHash, &s.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredReset{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) ConsumePasswordReset(ctx context.Context, resetID string) (bool, error) {
	tag, err := r.db.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, resetID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
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
	if len(s) <= n {
		return s
	}
	return s[:n]
}
