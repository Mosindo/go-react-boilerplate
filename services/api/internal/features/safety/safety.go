// Package safety implements blocking and reporting.
package safety

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validReasons = map[string]bool{"spam": true, "fake": true, "harassment": true, "inappropriate": true, "underage": true, "other": true}

type BlockedUser struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	PhotoURL  *string   `json:"photoUrl"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlockRequest struct {
	UserID string `json:"userId"`
}

type ReportRequest struct {
	UserID  string `json:"userId"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type Service struct{ db *pgxpool.Pool }

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

func (s *Service) userExists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

// Block hides both users from each other and dissolves any match. The
// blocked user is never told; the pair also stays out of discovery because the
// block is checked symmetrically everywhere.
func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return platformerrors.Invalid("you cannot block yourself")
	}
	ok, err := s.userExists(ctx, blockedID)
	if err != nil {
		return err
	}
	if !ok {
		return platformerrors.ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)`, blockerID, blockedID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := s.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''),
			(SELECT ph.id::text FROM photos ph WHERE ph.user_id = b.blocked_id ORDER BY ph.position LIMIT 1), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 200`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var u BlockedUser
		var photoID *string
		if err := rows.Scan(&u.ID, &u.FirstName, &photoID, &u.BlockedAt); err != nil {
			return nil, err
		}
		if photoID != nil {
			url := profiles.PhotoURL(*photoID)
			u.PhotoURL = &url
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	if reporterID == req.UserID {
		return platformerrors.Invalid("you cannot report yourself")
	}
	if !validReasons[req.Reason] {
		return platformerrors.Invalid("invalid reason")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return platformerrors.Invalid("details are too long")
	}
	ok, err := s.userExists(ctx, req.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return platformerrors.ErrNotFound
	}
	// At most one open report per reporter/target pair keeps the queue clean.
	_, err = s.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open')`,
		reporterID, req.UserID, req.Reason, details)
	return err
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if _, ok := httpx.UUIDParamValue(req.UserID); !ok {
		httpx.Fail(c, "safety.block", platformerrors.Invalid("invalid userId"))
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		httpx.Fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.ListBlocked(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "safety.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if _, ok := httpx.UUIDParamValue(req.UserID); !ok {
		httpx.Fail(c, "safety.report", platformerrors.Invalid("invalid userId"))
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = platformerrors.ErrNotFound
		}
		httpx.Fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusAccepted)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, reportLimit gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reportLimit, h.Report)
}
