// Package safety implements blocking and reporting.
package safety

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validReasons = map[string]bool{
	"spam": true, "fake_profile": true, "harassment": true,
	"inappropriate_content": true, "underage": true, "other": true,
}

type BlockedUser struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type ReportRequest struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type Service struct {
	pool *pgxpool.Pool
	hub  *realtime.Hub
}

func NewService(pool *pgxpool.Pool, hub *realtime.Hub) *Service {
	return &Service{pool: pool, hub: hub}
}

func (s *Service) userExists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

// Block hides both users from each other everywhere and deletes any match
// (and its messages) between them. It is idempotent.
func (s *Service) Block(ctx context.Context, me, target string) error {
	if me == target {
		return perr.BadRequest("invalid_target", "Vous ne pouvez pas vous bloquer vous-même.")
	}
	if ok, err := s.userExists(ctx, target); err != nil {
		return err
	} else if !ok {
		return perr.NotFound("Utilisateur introuvable.")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, me, target); err != nil {
		return err
	}
	var matchID *string
	err = tx.QueryRow(ctx, `DELETE FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid) RETURNING id`, me, target).Scan(&matchID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if matchID != nil {
		ev := realtime.Event{Type: "match.removed", Data: map[string]any{"matchId": *matchID}}
		s.hub.Publish(me, ev)
		s.hub.Publish(target, ev)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, me, target string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, me, target)
	return err
}

func (s *Service) List(ctx context.Context, me string) ([]BlockedUser, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''),
		       (SELECT id FROM photos ph WHERE ph.user_id = b.blocked_id ORDER BY position LIMIT 1)
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 200`, me)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var u BlockedUser
		var photoID *string
		if err := rows.Scan(&u.UserID, &u.FirstName, &photoID); err != nil {
			return nil, err
		}
		_ = photoID // the photo route refuses blocked users, so no URL is exposed
		out = append(out, u)
	}
	return out, rows.Err()
}

const maxReportsPerDay = 5

func (s *Service) Report(ctx context.Context, me, target string, req ReportRequest) error {
	if me == target {
		return perr.BadRequest("invalid_target", "Vous ne pouvez pas vous signaler vous-même.")
	}
	if !validReasons[req.Reason] {
		return perr.BadRequest("invalid_reason", "Motif de signalement invalide.")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return perr.BadRequest("invalid_details", "Les détails ne doivent pas dépasser 1000 caractères.")
	}
	if ok, err := s.userExists(ctx, target); err != nil {
		return err
	} else if !ok {
		return perr.NotFound("Utilisateur introuvable.")
	}
	var recent int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND created_at > NOW() - INTERVAL '1 day'`, me).Scan(&recent); err != nil {
		return err
	}
	if recent >= maxReportsPerDay {
		return perr.New(http.StatusTooManyRequests, "rate_limited", "Trop de signalements aujourd'hui.")
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)`, me, target, req.Reason, details)
	return err
}

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Block(c *gin.Context) {
	if err := h.svc.Block(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		perr.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.svc.Unblock(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		perr.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		perr.Respond(c, perr.BadRequest("invalid_request", "Requête invalide."))
		return
	}
	if err := h.svc.Report(c.Request.Context(), c.GetString("userID"), c.Param("id"), req); err != nil {
		perr.Respond(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.List)
	r.POST("/users/:id/block", requireUser, h.Block)
	r.DELETE("/users/:id/block", requireUser, h.Unblock)
	r.POST("/users/:id/report", requireUser, h.Report)
}
