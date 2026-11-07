// Package notifications exposes the in-app notification feed (new likes and matches).
// Rows are created by the discovery feature inside the swipe transaction.
package notifications

import (
	"context"
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	ActorID   *string    `json:"actorId"`
	ActorName *string    `json:"actorName"`
	PhotoURL  *string    `json:"photoUrl"`
	RefID     *string    `json:"refId"`
	ReadAt    *time.Time `json:"readAt"`
	CreatedAt time.Time  `json:"createdAt"`
	photoID   *string
}

type PhotoURLer interface{ URL(photoID string) string }

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT n.id, n.type, n.actor_id, pf.first_name,
		       (SELECT ph.id::text FROM photos ph WHERE ph.user_id = n.actor_id ORDER BY ph.position LIMIT 1),
		       n.ref_id, n.read_at, n.created_at
		FROM notifications n LEFT JOIN profiles pf ON pf.user_id = n.actor_id
		WHERE n.user_id = $1
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = n.actor_id)
		                                           OR (b.blocker_id = n.actor_id AND b.blocked_id = $1))
		ORDER BY n.created_at DESC, n.id LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.ActorID, &n.ActorName, &n.photoID, &n.RefID, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*)::int FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

var errNotFound = errors.New("notification not found")

func (r *Repository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}

type Handler struct {
	repo   *Repository
	photos PhotoURLer
}

func NewHandler(repo *Repository, photos PhotoURLer) *Handler {
	return &Handler{repo: repo, photos: photos}
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, h.List)
	r.POST("/notifications/read", requireUser, h.ReadAll)
	r.POST("/notifications/:id/read", requireUser, h.ReadOne)
}

func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, 30, 100)
	if !ok {
		return
	}
	uid := c.GetString("userID")
	list, err := h.repo.List(c.Request.Context(), uid, limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", 500, err)
		httpx.Internal(c)
		return
	}
	for i := range list {
		if list[i].photoID != nil {
			u := h.photos.URL(*list[i].photoID)
			list[i].PhotoURL = &u
		}
	}
	unread, err := h.repo.UnreadCount(c.Request.Context(), uid)
	if err != nil {
		logger.LogHandlerError(c, "notifications.unread", 500, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"notifications": list, "unreadCount": unread})
}

func (h *Handler) ReadAll(c *gin.Context) {
	if err := h.repo.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		logger.LogHandlerError(c, "notifications.read_all", 500, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ReadOne(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.NotFound(c)
		return
	}
	if err := h.repo.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, errNotFound) {
			httpx.NotFound(c)
			return
		}
		logger.LogHandlerError(c, "notifications.read", 500, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
