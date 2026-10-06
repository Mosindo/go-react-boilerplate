// Package account handles irreversible account operations.
package account

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var ErrWrongPassword = errors.New("wrong password")

type Blobs interface {
	Delete(key string) error
}

type Service struct {
	pool  *pgxpool.Pool
	blobs Blobs
}

func NewService(pool *pgxpool.Pool, blobs Blobs) *Service {
	return &Service{pool: pool, blobs: blobs}
}

// Delete permanently erases the account after re-checking the password.
// Foreign keys cascade to profile, preferences, photos, swipes, matches,
// conversations, messages, blocks, reports (as reporter or reported),
// notifications and sessions; the photo files are then removed from storage.
func (s *Service) Delete(ctx context.Context, userID, password string) error {
	var hash string
	if err := s.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWrongPassword
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return ErrWrongPassword
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Same lock as photo uploads: no photo can slip in between listing and deleting.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "photos:"+userID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, key)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Conversations are shared rows: delete the ones this user took part in
	// (messages cascade) before the user row so no orphan thread of the other
	// person's data lingers.
	if _, err := tx.Exec(ctx, `
		DELETE FROM conversations WHERE id IN (
			SELECT conversation_id FROM conversation_participants WHERE user_id = $1
		)
	`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	for _, key := range keys {
		if err := s.blobs.Delete(key); err != nil {
			log.Printf(`{"event":"account_blob_delete_failed","key":%q,"error":%q}`, key, err.Error())
		}
	}
	return nil
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type deleteRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

func (h *Handler) Delete(c *gin.Context) {
	var req deleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		if errors.Is(err, ErrWrongPassword) {
			httpx.Error(c, http.StatusForbidden, "wrong password")
			return
		}
		logger.LogHandlerError(c, "account.delete", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.DELETE("/account", requireUser, middleware.RateLimitByUser(5, time.Hour), h.Delete)
}
