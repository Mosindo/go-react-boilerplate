// Package notifications exposes in-app notifications (new matches). Push
// delivery (APNs/FCM) is out of scope; the realtime hub emits live events.
package notifications

import (
	"context"
	"net/http"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Notification struct {
	ID        string        `json:"id"`
	Type      string        `json:"type"`
	MatchID   *string       `json:"matchId"`
	Actor     *ActorSummary `json:"actor"`
	ReadAt    *time.Time    `json:"readAt"`
	CreatedAt time.Time     `json:"createdAt"`
}

type ActorSummary struct {
	ID        string  `json:"id"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, userID string, before time.Time, limit int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT n.id, n.type, n.match_id, n.read_at, n.created_at, a.user_id, a.first_name,
			(SELECT ph.id::text FROM photos ph WHERE ph.user_id = a.user_id ORDER BY ph.position LIMIT 1)
		FROM notifications n
		LEFT JOIN profiles a ON a.user_id = n.actor_id
		WHERE n.user_id = $1 AND n.created_at < $2
		ORDER BY n.created_at DESC, n.id
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var (
			n       Notification
			actorID *string
			name    *string
			photoID *string
		)
		if err := rows.Scan(&n.ID, &n.Type, &n.MatchID, &n.ReadAt, &n.CreatedAt, &actorID, &name, &photoID); err != nil {
			return nil, err
		}
		if actorID != nil && name != nil {
			actor := &ActorSummary{ID: *actorID, FirstName: *name}
			if photoID != nil {
				url := profiles.PhotoURL(*photoID)
				actor.PhotoURL = &url
			}
			n.Actor = actor
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *Repository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	return tag.RowsAffected() > 0, err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(c *gin.Context) {
	before := time.Now().Add(time.Minute)
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.Fail(c, "notifications.cursor", platformerrors.Invalid("before must be an RFC3339 timestamp"))
			return
		}
		before = t
	}
	items, err := h.repo.List(c.Request.Context(), httpx.UserID(c), before, httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	unread, err := h.repo.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "notifications.unread", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"notifications": items, "unreadCount": unread})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	found, err := h.repo.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "notifications.read", err)
		return
	}
	if !found {
		httpx.Fail(c, "notifications.read", platformerrors.ErrNotFound)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.repo.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, h.List)
	r.POST("/notifications/read", requireUser, h.MarkAllRead)
	r.POST("/notifications/:id/read", requireUser, h.MarkRead)
}
