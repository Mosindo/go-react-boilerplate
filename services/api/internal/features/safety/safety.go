// Package safety implements blocking and reporting of other users.
package safety

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/validate"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSelf       = errors.New("you cannot do this to yourself")
	ErrNotFound   = errors.New("user not found")
	ErrValidation = errors.New("validation error")
)

var validReasons = map[string]struct{}{
	"spam": {}, "fake": {}, "harassment": {}, "inappropriate": {}, "underage": {}, "other": {},
}

type BlockedUser struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=64"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=64"`
	Reason  string `json:"reason" binding:"required,max=32"`
	Details string `json:"details" binding:"max=4000"`
}

type Repository interface {
	Block(ctx context.Context, blocker, blocked string) error
	Unblock(ctx context.Context, blocker, blocked string) error
	ListBlocked(ctx context.Context, blocker string) ([]BlockedUser, error)
	Report(ctx context.Context, reporter, reported, reason, details string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (r *PGRepository) Block(ctx context.Context, blocker, blocked string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blocker, blocked)
	if isFKViolation(err) {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blocker string) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, '')
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 200
	`, blocker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BlockedUser, 0)
	for rows.Next() {
		var u BlockedUser
		if err := rows.Scan(&u.ID, &u.FirstName); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PGRepository) Report(ctx context.Context, reporter, reported, reason, details string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
	`, reporter, reported, reason, details)
	if isFKViolation(err) {
		return ErrNotFound
	}
	return err
}

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

// Block hides both users from each other everywhere (discovery, conversations, photos, profiles).
// The blocked user is not told.
func (s *Service) Block(ctx context.Context, blocker, blocked string) error {
	if blocker == blocked {
		return ErrSelf
	}
	if err := s.repo.Block(ctx, blocker, blocked); err != nil {
		return err
	}
	s.events.Publish(blocked, realtime.Event{Type: "conversations.changed"})
	return nil
}

func (s *Service) Unblock(ctx context.Context, blocker, blocked string) error {
	return s.repo.Unblock(ctx, blocker, blocked)
}

func (s *Service) ListBlocked(ctx context.Context, blocker string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, blocker)
}

func (s *Service) Report(ctx context.Context, reporter, reported, reason, details string) error {
	if reporter == reported {
		return ErrSelf
	}
	reason = strings.ToLower(strings.TrimSpace(reason))
	if _, ok := validReasons[reason]; !ok {
		return ErrValidation
	}
	details = strings.TrimSpace(details)
	if utf8.RuneCountInString(details) > 1000 {
		return ErrValidation
	}
	return s.repo.Report(ctx, reporter, reported, reason, details)
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelf):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason must be spam, fake, harassment, inappropriate, underage or other; details up to 1000 characters"})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil || !validate.IsUUID(req.UserID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id := c.Param("userId")
	if !validate.IsUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocked(c *gin.Context) {
	items, err := h.service.ListBlocked(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "safety.list_blocked", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil || !validate.IsUUID(req.UserID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req.UserID, req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusCreated)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, reportLimiter gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.ListBlocked)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reportLimiter, h.Report)
}
