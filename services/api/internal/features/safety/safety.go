// Package safety implements blocking and reporting. Blocking is symmetric in effect: neither user
// can see, like, match or message the other, and any existing match (with its messages) is removed.
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
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSelf     = errors.New("safety: cannot target yourself")
	ErrNoTarget = errors.New("safety: user not found")
)

var reportReasons = map[string]bool{"spam": true, "fake": true, "harassment": true, "inappropriate": true, "underage": true, "other": true}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	PhotoURL  *string   `json:"photoUrl"`
	BlockedAt time.Time `json:"blockedAt"`
	photoID   *string
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
	Block   bool   `json:"block"`
}

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) exists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

// Block inserts the block and removes any match between the pair, returning the removed match id (if any).
func (r *Repository) Block(ctx context.Context, blocker, blocked string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blocker, blocked); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return "", ErrNoTarget
		}
		return "", err
	}
	var matchID *string
	_ = tx.QueryRow(ctx, `
		DELETE FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		RETURNING id::text`, blocker, blocked).Scan(&matchID)
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if matchID == nil {
		return "", nil
	}
	return *matchID, nil
}

func (r *Repository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return err
}

func (r *Repository) ListBlocked(ctx context.Context, blocker string, limit, offset int) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(pf.first_name, ''),
		       (SELECT ph.id::text FROM photos ph WHERE ph.user_id = b.blocked_id ORDER BY ph.position LIMIT 1), b.created_at
		FROM blocks b LEFT JOIN profiles pf ON pf.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC, b.blocked_id LIMIT $2 OFFSET $3`, blocker, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.photoID, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) Report(ctx context.Context, reporter, reported, reason, details string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)`, reporter, reported, reason, details)
	return err
}

type PhotoURLer interface{ URL(photoID string) string }

type Handler struct {
	repo   *Repository
	rt     realtime.Publisher
	photos PhotoURLer
}

func NewHandler(repo *Repository, rt realtime.Publisher, photos PhotoURLer) *Handler {
	return &Handler{repo: repo, rt: rt, photos: photos}
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, reportLimiter *middleware.Limiter) {
	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, middleware.RateLimitByUser(reportLimiter), h.Report)
}

func (h *Handler) block(c *gin.Context, blocker, target string) bool {
	if blocker == target {
		httpx.BadRequest(c, "you cannot target yourself")
		return false
	}
	matchID, err := h.repo.Block(c.Request.Context(), blocker, target)
	if err != nil {
		if errors.Is(err, ErrNoTarget) {
			httpx.NotFound(c)
			return false
		}
		logger.LogHandlerError(c, "safety.block", 500, err)
		httpx.Internal(c)
		return false
	}
	if matchID != "" {
		h.rt.Publish(target, realtime.Event{Type: "unmatch", Data: map[string]string{"matchId": matchID}})
	}
	return true
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId is required")
		return
	}
	if h.block(c, c.GetString("userID"), req.UserID) {
		c.Status(http.StatusNoContent)
	}
}

func (h *Handler) Unblock(c *gin.Context) {
	id := c.Param("userId")
	if _, err := uuid.Parse(id); err != nil {
		httpx.NotFound(c)
		return
	}
	if err := h.repo.Unblock(c.Request.Context(), c.GetString("userID"), id); err != nil {
		logger.LogHandlerError(c, "safety.unblock", 500, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, 50, 100)
	if !ok {
		return
	}
	list, err := h.repo.ListBlocked(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "safety.list", 500, err)
		httpx.Internal(c)
		return
	}
	for i := range list {
		if list[i].photoID != nil {
			u := h.photos.URL(*list[i].photoID)
			list[i].PhotoURL = &u
		}
	}
	c.JSON(http.StatusOK, gin.H{"blocks": list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId and reason are required")
		return
	}
	details := strings.TrimSpace(req.Details)
	if !reportReasons[req.Reason] || utf8.RuneCountInString(details) > 1000 {
		httpx.BadRequest(c, "invalid reason or details (max 1000 characters)")
		return
	}
	me := c.GetString("userID")
	if me == req.UserID {
		httpx.BadRequest(c, "you cannot target yourself")
		return
	}
	ok, err := h.repo.exists(c.Request.Context(), req.UserID)
	if err != nil {
		logger.LogHandlerError(c, "safety.report", 500, err)
		httpx.Internal(c)
		return
	}
	if !ok {
		httpx.NotFound(c)
		return
	}
	if err := h.repo.Report(c.Request.Context(), me, req.UserID, req.Reason, details); err != nil {
		logger.LogHandlerError(c, "safety.report", 500, err)
		httpx.Internal(c)
		return
	}
	if req.Block && !h.block(c, me, req.UserID) {
		return
	}
	c.Status(http.StatusNoContent)
}
