// Package notifications exposes the in-app notification feed (matches and new messages).
package notifications

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Notification struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	ActorID        string     `json:"actorId"`
	ActorName      string     `json:"actorName"`
	ActorPhotoURL  string     `json:"actorPhotoUrl,omitempty"`
	MatchID        string     `json:"matchId"`
	ConversationID string     `json:"conversationId,omitempty"`
	ReadAt         *time.Time `json:"readAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id::text, n.type, n.actor_id::text, p.first_name,
		       (SELECT ph.id::text FROM photos ph WHERE ph.user_id = n.actor_id ORDER BY ph.position LIMIT 1),
		       n.match_id::text, c.id::text, n.read_at, n.created_at
		FROM notifications n
		JOIN profiles p ON p.user_id = n.actor_id
		JOIN matches m ON m.id = n.match_id AND m.unmatched_at IS NULL
		LEFT JOIN conversations c ON c.match_id = m.id
		WHERE n.user_id = $1
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = n.user_id AND b.blocked_id = n.actor_id)
		                                            OR (b.blocker_id = n.actor_id AND b.blocked_id = n.user_id))
		  AND ($2::timestamptz IS NULL OR n.created_at < $2)
		ORDER BY n.created_at DESC, n.id DESC LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		var photo, conv *string
		if err := rows.Scan(&n.ID, &n.Type, &n.ActorID, &n.ActorName, &photo, &n.MatchID, &conv, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		if photo != nil {
			n.ActorPhotoURL = "/photos/" + *photo
		}
		if conv != nil {
			n.ConversationID = *conv
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkRead marks one notification (id != "") or all of the user's notifications as read.
func (r *Repository) MarkRead(ctx context.Context, userID, id string) (int64, error) {
	if id == "" {
		tag, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
		return tag.RowsAffected(), err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $2 AND user_id = $1`, userID, id)
	return tag.RowsAffected(), err
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(c *gin.Context) {
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "before must be an RFC3339 timestamp")
			return
		}
		before = &t
	}
	limit := httpx.IntQuery(c, "limit", 30, 1, 100)
	uid := c.GetString("userID")
	list, err := h.repo.List(c.Request.Context(), uid, before, limit)
	if err != nil {
		h.fail(c, "notifications.list", err)
		return
	}
	unread, err := h.repo.UnreadCount(c.Request.Context(), uid)
	if err != nil {
		h.fail(c, "notifications.unread", err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, gin.H{"notifications": list, "unreadCount": unread})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	if !uuidRe.MatchString(id) {
		httpx.Error(c, http.StatusNotFound, "notification not found")
		return
	}
	n, err := h.repo.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "notifications.read", err)
		return
	}
	if n == 0 {
		httpx.Error(c, http.StatusNotFound, "notification not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if _, err := h.repo.MarkRead(c.Request.Context(), c.GetString("userID"), ""); err != nil {
		h.fail(c, "notifications.readall", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
	httpx.Error(c, http.StatusInternalServerError, "internal error")
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, h.List)
	r.POST("/notifications/read", requireUser, h.MarkAllRead)
	r.POST("/notifications/:id/read", requireUser, h.MarkRead)
}
