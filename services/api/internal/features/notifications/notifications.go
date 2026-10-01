// Package notifications stores in-app notifications and pushes them in real time.
package notifications

import (
	"context"
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	KindMatch   = "match"
	KindMessage = "message"
)

type Notification struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	RefID         string     `json:"refId"`
	ActorID       *string    `json:"actorId"`
	ActorName     string     `json:"actorName"`
	ActorThumbURL *string    `json:"actorThumbUrl"`
	IsRead        bool       `json:"isRead"`
	CreatedAt     time.Time  `json:"createdAt"`
	ReadAt        *time.Time `json:"readAt"`
}

type Publisher interface {
	Publish(userID string, ev realtime.Event)
}

type Service struct {
	pool *pgxpool.Pool
	pub  Publisher
}

func NewService(pool *pgxpool.Pool, pub Publisher) *Service { return &Service{pool: pool, pub: pub} }

// Notify creates (or, for repeated message notifications, refreshes) an unread notification and
// tells connected clients to update their badge.
func (s *Service) Notify(ctx context.Context, userID, kind, actorID, refID string) error {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, type, ref_id) WHERE read_at IS NULL
		DO UPDATE SET created_at = NOW(), actor_id = EXCLUDED.actor_id
		RETURNING id
	`, userID, kind, actorID, refID).Scan(&id)
	if err != nil {
		return err
	}
	s.pub.Publish(userID, realtime.Event{Type: "notification.new", Data: map[string]string{"id": id, "type": kind, "refId": refID}})
	return nil
}

// MarkRefRead marks unread notifications of a kind about ref (e.g. a conversation) as read.
func (s *Service) MarkRefRead(ctx context.Context, userID, kind, refID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND type = $2 AND ref_id = $3 AND read_at IS NULL`, userID, kind, refID)
	return err
}

func (s *Service) List(ctx context.Context, userID string, limit int, before *time.Time) ([]Notification, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT n.id, n.type, n.ref_id, n.actor_id, COALESCE(p.first_name, ''),
		       CASE WHEN ph.id IS NOT NULL THEN '/photos/' || ph.id || '/thumb' END,
		       n.read_at IS NOT NULL, n.created_at, n.read_at
		FROM notifications n
		LEFT JOIN profiles p ON p.user_id = n.actor_id
		LEFT JOIN photos ph ON ph.user_id = n.actor_id AND ph.position = 0
		WHERE n.user_id = $1 AND ($3::timestamptz IS NULL OR n.created_at < $3)
		  AND (n.actor_id IS NULL OR NOT EXISTS (
		        SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = n.actor_id) OR (b.blocker_id = n.actor_id AND b.blocked_id = $1)))
		ORDER BY n.created_at DESC, n.id DESC LIMIT $2
	`, userID, limit, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.RefID, &n.ActorID, &n.ActorName, &n.ActorThumbURL, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkRead marks one notification read; it reports false when it is not the caller's.
func isUUIDError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $2 AND user_id = $1`, userID, id)
	if err != nil {
		if isUUIDError(err) {
			return false, nil
		}
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	before, ok, err := httpx.Cursor(c, "before")
	if err != nil {
		httpx.BadRequest(c, "curseur invalide")
		return
	}
	var cursor *time.Time
	if ok {
		cursor = &before
	}
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 30, 100), cursor)
	if err != nil {
		httpx.Internal(c, "notifications.list", err)
		return
	}
	unread, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "notifications.unread", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": items, "unreadCount": unread})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "notifications.unread", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	ok, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Internal(c, "notifications.read", err)
		return
	}
	if !ok {
		httpx.Error(c, http.StatusNotFound, "not_found", "notification introuvable")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Internal(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	limit := middleware.NewLimiter(300, time.Minute).ByUser()
	r.GET("/notifications", requireUser, limit, h.List)
	r.GET("/notifications/unread-count", requireUser, limit, h.UnreadCount)
	r.POST("/notifications/read-all", requireUser, h.MarkAllRead)
	r.POST("/notifications/:id/read", requireUser, h.MarkRead)
}
