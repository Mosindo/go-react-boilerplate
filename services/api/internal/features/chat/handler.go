package chat

import (
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Conversations(c *gin.Context) {
	limit, err := pagination.Limit(c.Query("limit"), 30, 50)
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	res, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"))
	h.reply(c, http.StatusOK, res, err)
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	limit, err := pagination.Limit(c.Query("limit"), 40, 100)
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	res, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), id, limit, c.Query("cursor"))
	h.reply(c, http.StatusOK, res, err)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), id, req.Body)
	h.reply(c, http.StatusCreated, msg, err)
}

func (h *Handler) Read(c *gin.Context) {
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

func (h *Handler) Hide(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), id); err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unread(c *gin.Context) {
	res, err := h.service.Unread(c.Request.Context(), c.GetString("userID"))
	h.reply(c, http.StatusOK, res, err)
}

func (h *Handler) reply(c *gin.Context, status int, body any, err error) {
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, body)
}
