// Package account handles self-service account operations: password change and deletion.
package account

import (
	"context"
	"errors"
	"net/http"

	"example.com/api/internal/features/auth"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/media"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type DeleteRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=128"`
	NewPassword     string `json:"newPassword" binding:"required,max=128"`
}

type Repository interface {
	PasswordHash(ctx context.Context, userID string) (string, error)
	PhotoKeys(ctx context.Context, userID string) ([]string, error)
	DeleteUser(ctx context.Context, userID string) error
	SetPassword(ctx context.Context, userID, hash, keepSessionID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) PasswordHash(ctx context.Context, userID string) (string, error) {
	var h string
	err := r.db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&h)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("user not found")
	}
	return h, err
}

func (r *PGRepository) PhotoKeys(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// DeleteUser removes the user; every dependent row (profile, photos, swipes,
// matches, conversations, messages, blocks, reports, notifications, sessions)
// is removed by ON DELETE CASCADE.
func (r *PGRepository) DeleteUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}

func (r *PGRepository) SetPassword(ctx context.Context, userID, hash, keepSessionID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, hash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL`, userID, keepSessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type Service struct {
	repo  Repository
	store *media.Store
}

func NewService(repo Repository, store *media.Store) *Service {
	return &Service{repo: repo, store: store}
}

var errWrongPassword = platformerrors.New(http.StatusForbidden, "invalid_credentials", "wrong password")

func (s *Service) verify(ctx context.Context, userID, password string) error {
	hash, err := s.repo.PasswordHash(ctx, userID)
	if err != nil {
		return platformerrors.New(http.StatusUnauthorized, "unauthorized", "user not found")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return errWrongPassword
	}
	return nil
}

// Delete permanently removes the account and its files after re-authentication.
func (s *Service) Delete(ctx context.Context, userID, password string) error {
	if err := s.verify(ctx, userID, password); err != nil {
		return err
	}
	keys, err := s.repo.PhotoKeys(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}
	for _, k := range keys {
		_ = s.store.Delete(k)
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, sessionID, current, next string) error {
	if err := s.verify(ctx, userID, current); err != nil {
		return err
	}
	if err := auth.ValidatePassword(next); err != nil {
		return platformerrors.New(http.StatusBadRequest, "validation", err.Error())
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.SetPassword(ctx, userID, string(hash), sessionID)
}

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		httpx.Fail(c, "account.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword); err != nil {
		httpx.Fail(c, "account.password", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// RegisterRoutes mounts account routes behind the strict limiter (password checks).
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, strict gin.HandlerFunc) {
	g := r.Group("/me", requireUser, strict)
	g.DELETE("", h.Delete)
	g.PUT("/password", h.ChangePassword)
}
