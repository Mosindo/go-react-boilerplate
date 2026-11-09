// Package users handles account-level operations (permanent deletion).
package users

import (
	"context"
	"errors"
	"log"
	"net/http"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var ErrWrongPassword = errors.New("password is incorrect")

type DeleteRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type Service struct {
	pool    *pgxpool.Pool
	storage photos.Storage
}

func NewService(pool *pgxpool.Pool, storage photos.Storage) *Service {
	return &Service{pool: pool, storage: storage}
}

// DeleteAccount verifies the password, then deletes the user. Foreign keys cascade to
// profile, preferences, interests, photos rows, swipes, matches, conversations, messages,
// notifications, blocks, reports and sessions. Photo files are removed from storage afterwards.
func (s *Service) DeleteAccount(ctx context.Context, userID, password string) error {
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

	rows, err := s.pool.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return err
	}
	for _, k := range keys {
		if err := s.storage.Delete(k); err != nil {
			log.Printf("users: could not delete photo file %s: %v", k, err)
		}
	}
	return nil
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "password is required")
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		if errors.Is(err, ErrWrongPassword) {
			httpx.Error(c, http.StatusForbidden, err.Error())
			return
		}
		logger.LogHandlerError(c, "users.delete", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, limit gin.HandlerFunc) {
	r.DELETE("/me", requireUser, limit, h.Delete)
}
