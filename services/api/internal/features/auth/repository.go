package auth

import (
	"context"
	"errors"
	"net"
	"time"

	"example.com/api/internal/platform/db"
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
	ID              string
	Email           string
	ProfileComplete bool
	CreatedAt       time.Time
}

type StoredUserWithPassword struct {
	User         StoredUser
	PasswordHash string
}

type StoredSession struct {
	ID     string
	UserID string
}

type StoredReset struct {
	ID        string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
}

type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (StoredUser, error)
	GetUserAuthByEmail(ctx context.Context, email string) (StoredUserWithPassword, error)
	GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error)
	GetUserByID(ctx context.Context, userID string) (StoredUser, error)
	CreateSession(ctx context.Context, userID, tokenHash, userAgent, ipAddress string, expiresAt time.Time) (StoredSession, error)
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (StoredSession, error)
	RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error)
	RevokeSessionByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
	TouchUser(ctx context.Context, userID string) error

	CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time, cooldown time.Duration) (bool, error)
	LatestPasswordReset(ctx context.Context, userID string) (StoredReset, error)
	IncrementResetAttempts(ctx context.Context, resetID string) error
	CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error
	UpdatePassword(ctx context.Context, userID, passwordHash, keepSessionID string) error
	DeleteUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

var userColumns = `u.id, u.email, u.created_at, ` + db.ProfileComplete("u.id")

func scanUser(row pgx.Row, extra ...any) (StoredUser, error) {
	var user StoredUser
	dest := append([]any{&user.ID, &user.Email, &user.CreatedAt, &user.ProfileComplete}, extra...)
	err := row.Scan(dest...)
	return user, err
}

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
	var hash string
	user, err := scanUser(r.dbPool.QueryRow(ctx, `
		SELECT `+userColumns+`, u.password_hash FROM users u WHERE u.email = $1
	`, email), &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return StoredUserWithPassword{User: user, PasswordHash: hash}, nil
}

func (r *PGRepository) GetUserAuthByID(ctx context.Context, userID string) (StoredUserWithPassword, error) {
	var hash string
	user, err := scanUser(r.dbPool.QueryRow(ctx, `
		SELECT `+userColumns+`, u.password_hash FROM users u WHERE u.id = $1
	`, userID), &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUserWithPassword{}, ErrRepositoryNotFound
		}
		return StoredUserWithPassword{}, err
	}
	return StoredUserWithPassword{User: user, PasswordHash: hash}, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, userID string) (StoredUser, error) {
	user, err := scanUser(r.dbPool.QueryRow(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id = $1`, userID))
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
		SELECT u.id, $2, $3, $4, $5, NOW() FROM users u WHERE u.id = $1
		RETURNING id, user_id
	`, userID, tokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt).Scan(&session.ID, &session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSession{}, ErrRepositoryNotFound
		}
		return StoredSession{}, err
	}
	return session, nil
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

func (r *PGRepository) RotateSessionToken(ctx context.Context, sessionID, nextTokenHash, userAgent, ipAddress string, expiresAt, usedAt time.Time) (StoredSession, error) {
	var session StoredSession
	err := r.dbPool.QueryRow(ctx, `
		UPDATE sessions
		SET token_hash = $2,
		    user_agent = COALESCE($3, user_agent),
		    ip_address = COALESCE($4, ip_address),
		    expires_at = $5,
		    last_used_at = $6
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id
	`, sessionID, nextTokenHash, nullableString(userAgent), nullableIP(ipAddress), expiresAt, usedAt).Scan(&session.ID, &session.UserID)
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

func (r *PGRepository) IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error) {
	var active bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)
	`, sessionID, userID).Scan(&active)
	return active, err
}

func (r *PGRepository) TouchUser(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)
	return err
}

// CreatePasswordReset invalidates older codes and stores the new one. It returns false (and stores
// nothing) when a code was already requested within the cooldown, which prevents mail bombing.
func (r *PGRepository) CreatePasswordReset(ctx context.Context, userID, codeHash string, expiresAt time.Time, cooldown time.Duration) (bool, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var recent bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM password_resets WHERE user_id = $1 AND created_at > NOW() - make_interval(secs => $2))
	`, userID, cooldown.Seconds()).Scan(&recent); err != nil {
		return false, err
	}
	if recent {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE password_resets SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_resets (user_id, code_hash, expires_at) VALUES ($1, $2, $3)
	`, userID, codeHash, expiresAt); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (r *PGRepository) LatestPasswordReset(ctx context.Context, userID string) (StoredReset, error) {
	var reset StoredReset
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, code_hash, attempts, expires_at FROM password_resets
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&reset.ID, &reset.CodeHash, &reset.Attempts, &reset.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredReset{}, ErrRepositoryNotFound
		}
		return StoredReset{}, err
	}
	return reset, nil
}

func (r *PGRepository) IncrementResetAttempts(ctx context.Context, resetID string) error {
	_, err := r.dbPool.Exec(ctx, `UPDATE password_resets SET attempts = attempts + 1 WHERE id = $1`, resetID)
	return err
}

func (r *PGRepository) CompletePasswordReset(ctx context.Context, resetID, userID, passwordHash string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

// UpdatePassword changes the hash and revokes every session except keepSessionID.
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
		WHERE user_id = $1 AND revoked_at IS NULL AND id <> $2
	`, userID, keepSessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) DeleteUser(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableIP(value string) any {
	if value == "" {
		return nil
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	return ip.String()
}
