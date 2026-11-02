// Package notifications stores in-app notifications and pushes them live.
package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Notification struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Read      bool            `json:"read"`
	CreatedAt time.Time       `json:"createdAt"`
}

type Service struct {
	pool *pgxpool.Pool
	hub  *realtime.Hub
}

func NewService(pool *pgxpool.Pool, hub *realtime.Hub) *Service {
	return &Service{pool: pool, hub: hub}
}

// Create stores a notification and pushes it to the user's live connections.
// For message notifications, an existing unread one for the same match is
// reused so a long conversation does not flood the list.
func (s *Service) Create(ctx context.Context, userID, typ string, data map[string]any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var n Notification
	var readAt *time.Time
	if typ == TypeMessage {
		err = s.pool.QueryRow(ctx, `
			UPDATE notifications SET data = $3, created_at = NOW()
			WHERE id = (SELECT id FROM notifications WHERE user_id = $1 AND type = 'message' AND read_at IS NULL
			            AND data->>'matchId' = $4 LIMIT 1)
			RETURNING id, type, data, read_at, created_at`, userID, typ, payload, data["matchId"]).
			Scan(&n.ID, &n.Type, &n.Data, &readAt, &n.CreatedAt)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
	}
	if n.ID == "" {
		err = s.pool.QueryRow(ctx, `
			INSERT INTO notifications (user_id, type, data) VALUES ($1, $2, $3)
			RETURNING id, type, data, read_at, created_at`, userID, typ, payload).
			Scan(&n.ID, &n.Type, &n.Data, &readAt, &n.CreatedAt)
		if err != nil {
			return err
		}
	}
	n.Read = readAt != nil
	s.hub.Publish(userID, realtime.Event{Type: "notification.new", Data: n})
	return nil
}

type listResponse struct {
	Items       []Notification `json:"items"`
	UnreadCount int            `json:"unreadCount"`
	NextCursor  *string        `json:"nextCursor"`
}

func (s *Service) List(ctx context.Context, userID string, limit int, before *time.Time) (listResponse, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, type, data, read_at IS NOT NULL, created_at FROM notifications
		WHERE user_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC, id DESC LIMIT $3`, userID, before, limit+1)
	if err != nil {
		return listResponse{}, err
	}
	defer rows.Close()
	out := listResponse{Items: []Notification{}}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.Data, &n.Read, &n.CreatedAt); err != nil {
			return out, err
		}
		out.Items = append(out.Items, n)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		c := out.Items[limit-1].CreatedAt.Format(time.RFC3339Nano)
		out.NextCursor = &c
	}
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&out.UnreadCount)
	return out, err
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	return tag.RowsAffected() > 0, err
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *gin.Context) {
	limit := 30
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			perr.Respond(c, perr.BadRequest("invalid_limit", "limit doit être entre 1 et 100."))
			return
		}
		limit = n
	}
	var before *time.Time
	if v := c.Query("cursor"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			perr.Respond(c, perr.BadRequest("invalid_cursor", "Curseur invalide."))
			return
		}
		before = &t
	}
	res, err := h.svc.List(c.Request.Context(), c.GetString("userID"), limit, before)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) MarkRead(c *gin.Context) {
	ok, err := h.svc.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		perr.Respond(c, perr.NotFound("Notification introuvable."))
		return
	}
	if !ok {
		perr.Respond(c, perr.NotFound("Notification introuvable."))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.svc.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		perr.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, h.List)
	r.POST("/notifications/read-all", requireUser, h.MarkAllRead)
	r.POST("/notifications/:id/read", requireUser, h.MarkRead)
}
