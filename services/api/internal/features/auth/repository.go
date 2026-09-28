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
	ErrRepositoryResetGone   = errors.New("auth reset not usable")
)

type StoredUser struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type StoredSession struct {
	ID     string
	UserID string
}

type StoredReset struct {
	ID       string
	CodeHash string
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserByEmail(ctx context.Context, email string) (StoredUser, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) error
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error
	UpdatePasswordRevokeOthers(ctx context.Context, userID, passwordHash, keepSessionID string) error
	DeleteUser(ctx context.Context, userID string) error
	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error
	ClaimResetAttempt(ctx context.Context, userID string, maxAttempts int) (StoredReset, error)
	CompleteReset(ctx context.Context, resetID, userID, passwordHash string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StoredUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var u StoredUser
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at
	`, email, passwordHash).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrRepositoryEmailExists
		}
		return StoredUser{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, u.ID); err != nil {
		return StoredUser{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoredUser{}, err
	}
	return u, nil
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (StoredUser, error) {
	return r.scanUser(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`, email)
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	return r.scanUser(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE id = $1`, userID)
}

func (r *PGRepository) scanUser(ctx context.Context, query string, arg any) (StoredUser, error) {
	var u StoredUser
	err := r.pool.QueryRow(ctx, query, arg).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredUser{}, ErrRepositoryNotFound
	}
	return u, err
}

func (r *PGRepository) CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error) {
	var s StoredSession
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id
	`, userID, tokenHash, truncated(userAgent, 256), nullableIP(ipAddress), expiresAt).Scan(&s.ID, &s.UserID)
	return s, err
}

func (r *PGRepository) GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error) {
	var s StoredSession
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.user_id
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW()
	`, tokenHash).Scan(&s.ID, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredSession{}, ErrRepositorySessionGone
	}
	return s, err
}

// RotateSessionToken swaps the refresh token only if it still equals the one the
// caller presented, so two concurrent refreshes with the same token cannot both win.
func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, currentTokenHash, nextTokenHash, userAgent, ipAddress string, expiresAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE sessions
		SET token_hash = $3,
		    user_agent = COALESCE($4, user_agent),
		    ip_address = COALESCE($5, ip_address),
		    expires_at = $6,
		    last_used_at = NOW()
		WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > NOW()
	`, sessionID, currentTokenHash, nextTokenHash, truncated(userAgent, 256), nullableIP(ipAddress), expiresAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositorySessionGone
	}
	return nil
}

func (r *PGRepository) RevokeSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW(), last_used_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

// UpdatePasswordRevokeOthers sets the new hash and revokes every session except
// keepSessionID (pass "" to revoke all) in one transaction.
func (r *PGRepository) UpdatePasswordRevokeOthers(ctx context.Context, userID, passwordHash, keepSessionID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := setPasswordAndRevoke(ctx, tx, userID, passwordHash, keepSessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func setPasswordAndRevoke(ctx context.Context, tx pgx.Tx, userID, passwordHash, keepSessionID string) error {
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if keepSessionID == "" {
		_, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL`, userID, keepSessionID)
	return err
}

// DeleteUser hard-deletes the account; every dependent table cascades. Other
// users' notifications that merely point at the deleted user are removed too.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE data ->> 'userId' = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CreatePasswordReset replaces any earlier outstanding reset of the user.
func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM password_resets WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, codeHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ClaimResetAttempt atomically burns one guess of the newest usable reset and
// returns its hash for comparison. Because the counter is incremented before the
// comparison and in a single statement, parallel guesses cannot exceed the cap.
func (r *PGRepository) ClaimResetAttempt(ctx context.Context, userID string, maxAttempts int) (StoredReset, error) {
	var s StoredReset
	err := r.pool.QueryRow(ctx, `
		UPDATE password_resets
		SET attempts = attempts + 1
		WHERE id = (
			SELECT id FROM password_resets
			WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW() AND attempts < $2
			ORDER BY created_at DESC LIMIT 1
		)
		AND used_at IS NULL AND attempts < $2
		RETURNING id, code_hash
	`, userID, maxAttempts).Scan(&s.ID, &s.CodeHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredReset{}, ErrRepositoryResetGone
	}
	return s, err
}

// CompleteReset consumes the reset (single use), sets the password and revokes
// all sessions in one transaction.
func (r *PGRepository) CompleteReset(ctx context.Context, resetID, userID, passwordHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, resetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryResetGone
	}
	if err := setPasswordAndRevoke(ctx, tx, userID, passwordHash, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func truncated(v string, n int) any {
	if v == "" {
		return nil
	}
	if len(v) > n {
		v = v[:n]
	}
	// drop a possibly split multi-byte rune / invalid bytes
	return string([]rune(v))
}

func nullableIP(value string) any {
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.String()
}
