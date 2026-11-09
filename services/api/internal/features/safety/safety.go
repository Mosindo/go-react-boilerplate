// Package safety implements blocking and reporting.
package safety

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
)

var (
	ErrSelf          = errors.New("you cannot target yourself")
	ErrUserNotFound  = errors.New("user not found")
	ErrInvalidReason = errors.New("reason must be one of spam, fake, harassment, inappropriate, underage, other")
	ErrDetailsLong   = errors.New("details must be at most 1000 characters")
)

var reasons = map[string]bool{"spam": true, "fake": true, "harassment": true, "inappropriate": true, "underage": true, "other": true}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type ReportRequest struct {
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
	Block   bool   `json:"block"`
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Block records the block and ends any active match, so the conversation disappears for both users.
func (r *Repository) Block(ctx context.Context, blocker, blocked string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blocker, blocked); err != nil {
		if isFKViolation(err) {
			return ErrUserNotFound
		}
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE matches SET unmatched_at = NOW()
		WHERE unmatched_at IS NULL AND user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)`, blocker, blocked); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return err
}

func (r *Repository) List(ctx context.Context, blocker string) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id::text, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 200`, blocker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) Report(ctx context.Context, reporter, reported, reason, details string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)`,
		reporter, reported, reason, details)
	if isFKViolation(err) {
		return ErrUserNotFound
	}
	return err
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

type Service struct {
	repo *Repository
	rt   realtime.Publisher
}

func NewService(repo *Repository, rt realtime.Publisher) *Service {
	return &Service{repo: repo, rt: rt}
}

func (s *Service) check(actor, target string) error {
	if actor == target {
		return ErrSelf
	}
	if !uuidRe.MatchString(target) {
		return ErrUserNotFound
	}
	return nil
}

func (s *Service) Block(ctx context.Context, actor, target string) error {
	if err := s.check(actor, target); err != nil {
		return err
	}
	if err := s.repo.Block(ctx, actor, target); err != nil {
		return err
	}
	s.rt.Publish(target, realtime.Event{Type: "unmatch", Data: map[string]string{}})
	return nil
}

func (s *Service) Unblock(ctx context.Context, actor, target string) error {
	if err := s.check(actor, target); err != nil {
		return err
	}
	return s.repo.Unblock(ctx, actor, target)
}

func (s *Service) List(ctx context.Context, actor string) ([]BlockedUser, error) {
	return s.repo.List(ctx, actor)
}

func (s *Service) Report(ctx context.Context, actor, target string, req ReportRequest) error {
	if err := s.check(actor, target); err != nil {
		return err
	}
	req.Reason = strings.ToLower(strings.TrimSpace(req.Reason))
	if !reasons[req.Reason] {
		return ErrInvalidReason
	}
	req.Details = strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(req.Details) > 1000 {
		return ErrDetailsLong
	}
	if err := s.repo.Report(ctx, actor, target, req.Reason, req.Details); err != nil {
		return err
	}
	if req.Block {
		return s.Block(ctx, actor, target)
	}
	return nil
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelf), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsLong):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusNotFound, "user not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) Block(c *gin.Context) {
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "safety.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), c.Param("id"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusCreated)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, reportLimit gin.HandlerFunc) {
	r.GET("/me/blocks", requireUser, h.List)
	r.PUT("/users/:id/block", requireUser, h.Block)
	r.DELETE("/users/:id/block", requireUser, h.Unblock)
	r.POST("/users/:id/report", requireUser, reportLimit, h.Report)
}
