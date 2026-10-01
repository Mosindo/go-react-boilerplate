// Package moderation implements user blocking and reporting.
package moderation

import (
	"context"
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrSelf          = errors.New("cannot target yourself")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrDetailsLong   = errors.New("details too long")
)

var reasons = map[string]bool{"spam": true, "fake": true, "harassment": true, "inappropriate": true, "underage": true, "other": true}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID    string `json:"userId" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
	Details   string `json:"details"`
	AlsoBlock bool   `json:"alsoBlock"`
}

type Publisher interface {
	Publish(userID string, ev realtime.Event)
}

type Service struct {
	pool *pgxpool.Pool
	pub  Publisher
}

func NewService(pool *pgxpool.Pool, pub Publisher) *Service { return &Service{pool: pool, pub: pub} }

func uuidError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "22P02" || pgErr.Code == "23503")
}

func (s *Service) userExists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1::uuid)`, id).Scan(&ok)
	if uuidError(err) {
		return false, nil
	}
	return ok, err
}

// Block hides both users from each other everywhere and removes any match (with its conversation).
func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelf
	}
	ok, err := s.userExists(ctx, blockedID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)`, blockerID, blockedID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE (user_id = $1 AND actor_id = $2) OR (user_id = $2 AND actor_id = $1)`, blockerID, blockedID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		s.pub.Publish(blockedID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": blockerID}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	if err != nil {
		if uuidError(err) {
			return ErrNotFound
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 500
	`, blockerID)
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

// Report files a moderation report. Repeating the same open report within 24h is idempotent.
func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	if reporterID == req.UserID {
		return ErrSelf
	}
	if !reasons[req.Reason] {
		return ErrInvalidReason
	}
	details := profiles.CleanText(req.Details)
	if len([]rune(details)) > 1000 {
		return ErrDetailsLong
	}
	ok, err := s.userExists(ctx, req.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
		  SELECT 1 FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND reason = $3
		    AND status = 'open' AND created_at > NOW() - INTERVAL '24 hours')
	`, reporterID, req.UserID, req.Reason, details)
	if err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, reporterID, req.UserID)
	}
	return nil
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		httpx.Error(c, http.StatusNotFound, "not_found", "utilisateur introuvable")
	case errors.Is(err, ErrSelf):
		httpx.BadRequest(c, "action impossible sur votre propre compte")
	case errors.Is(err, ErrInvalidReason):
		httpx.BadRequest(c, "motif de signalement invalide")
	case errors.Is(err, ErrDetailsLong):
		httpx.BadRequest(c, "détails trop longs (1000 caractères maximum)")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId requis")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		fail(c, "moderation.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), c.Param("userId")); err != nil {
		fail(c, "moderation.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.ListBlocked(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		fail(c, "moderation.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId et motif requis")
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req); err != nil {
		fail(c, "moderation.report", err)
		return
	}
	c.Status(http.StatusAccepted)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	reportLimit := middleware.NewLimiter(20, time.Hour).ByUser()
	blockLimit := middleware.NewLimiter(120, time.Hour).ByUser()
	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, blockLimit, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reportLimit, h.Report)
}
