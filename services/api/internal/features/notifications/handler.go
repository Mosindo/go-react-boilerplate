package notifications

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ repo Repository }

func NewHandler(repo Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("userID")
	limit := httpx.Limit(c, 20, 50)
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.BadRequest(c, "notifications.list", "invalid before cursor", err)
			return
		}
		before = &t
	}
	items, err := h.repo.List(c.Request.Context(), userID, before, limit+1)
	if err != nil {
		httpx.Internal(c, "notifications.list", err)
		return
	}
	resp := ListResponse{}
	if len(items) > limit {
		items = items[:limit]
		next := items[len(items)-1].CreatedAt
		resp.NextBefore = &next
	}
	resp.Notifications = items
	if resp.UnreadCount, err = h.repo.UnreadCount(c.Request.Context(), userID); err != nil {
		httpx.Internal(c, "notifications.unread", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	err := h.repo.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, "notifications.read", http.StatusNotFound, "notification not found", nil)
	default:
		httpx.Internal(c, "notifications.read", err)
	}
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.repo.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		httpx.Internal(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
