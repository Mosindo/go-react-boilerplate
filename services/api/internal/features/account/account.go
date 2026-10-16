// Package account owns irreversible account operations (deletion).
package account

import (
	"context"
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var ErrWrongPassword = errors.New("password is incorrect")

// Media lists and purges a user's stored files around the database deletion.
type Media interface {
	StorageKeys(ctx context.Context, userID string) ([]string, error)
	PurgeObjects(keys []string)
}

// Sessions is told to drop live connections once the account is gone.
type Sessions interface {
	Kick(userID string)
}

type Service struct {
	db       *pgxpool.Pool
	media    Media
	sessions Sessions
}

func NewService(db *pgxpool.Pool, media Media, sessions Sessions) *Service {
	return &Service{db: db, media: media, sessions: sessions}
}

// Delete permanently removes the user after re-checking their password. Every table
// references users with ON DELETE CASCADE, so profile, photos, swipes, matches,
// messages, blocks, reports, notifications and sessions disappear in one statement;
// stored image files are removed right after.
func (s *Service) Delete(ctx context.Context, userID, password string) error {
	var hash string
	if err := s.db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already gone: deletion is idempotent
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return ErrWrongPassword
	}

	keys, err := s.media.StorageKeys(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return err
	}
	s.media.PurgeObjects(keys)
	s.sessions.Kick(userID)
	return nil
}

type DeleteRequest struct {
	Password string `json:"password" binding:"required,max=72"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, "account.delete.bind", http.StatusBadRequest, "password confirmation required", err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		if errors.Is(err, ErrWrongPassword) {
			httpx.Fail(c, "account.delete", http.StatusForbidden, "password is incorrect", err)
			return
		}
		httpx.Fail(c, "account.delete", http.StatusInternalServerError, "could not delete account", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.DELETE("/me", requireUser, middleware.NewLimiter(5, time.Hour).ByUser(), h.Delete)
}
