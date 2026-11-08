// Package safety implements blocking and reporting.
//
// A block is symmetric in effect: neither user can see the other in discovery, matches, chat,
// notifications or profile views. Blocking is reversible by the blocker; matches and messages are
// kept (only hidden) so that a moderator can still review a report.
package safety

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/validate"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ReportReasons = []string{"spam", "fake", "harassment", "inappropriate", "underage", "other"}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type userRequest struct {
	UserID string `json:"userId"`
}

type reportRequest struct {
	UserID  string `json:"userId"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

func (s *Service) checkTarget(ctx context.Context, userID, targetID string) error {
	if !profiles.IsUUID(targetID) {
		return platformerrors.Invalid("invalid userId")
	}
	if userID == targetID {
		return platformerrors.Invalid("you cannot target yourself")
	}
	var exists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, targetID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return platformerrors.ErrNotFound
	}
	return nil
}

// Block is idempotent.
func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if err := s.checkTarget(ctx, userID, targetID); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, targetID)
	return err
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	if !profiles.IsUUID(targetID) {
		return platformerrors.ErrNotFound
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, userID, targetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.ErrNotFound
	}
	return nil
}

func (s *Service) ListBlocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	rows, err := s.db.Query(ctx, `
		SELECT b.blocked_id, p.first_name, b.created_at FROM blocks b
		JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 200`, userID)
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

func (s *Service) Report(ctx context.Context, userID string, req reportRequest) error {
	if !slices.Contains(ReportReasons, req.Reason) {
		return platformerrors.Invalid("invalid reason")
	}
	details, ok := validate.CleanText(req.Details, 1000, true)
	if !ok {
		return platformerrors.Invalid("details must be at most 1000 characters")
	}
	if err := s.checkTarget(ctx, userID, req.UserID); err != nil {
		return err
	}
	// Identical open reports from the same person are merged instead of piling up.
	_, err := s.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM reports r WHERE r.reporter_id = $1 AND r.reported_id = $2 AND r.reason = $3 AND r.status = 'open')`,
		userID, req.UserID, req.Reason, details)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return platformerrors.Invalid("invalid report")
	}
	return err
}

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Block(c *gin.Context) {
	var req userRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID); err != nil {
		httpx.Fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), c.Param("userId")); err != nil {
		httpx.Fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.ListBlocked(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "safety.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		httpx.Fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusCreated)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, reports *middleware.Limiter) {
	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reports.ByUser(), h.Report)
}
