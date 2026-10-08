package notifications

import (
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, err := pagination.Limit(c.Query("limit"), 30, 50)
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	res, err := h.service.List(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"), c.Query("unread") == "true")
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, UnreadCountResponse{Unread: n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
