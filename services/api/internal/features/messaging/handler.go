package messaging

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "conversation not found")
	case errors.Is(err, ErrValidation):
		httpx.Fail(c, http.StatusUnprocessableEntity, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) List(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			httpx.Fail(c, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = n
	}
	before := c.Query("before")
	if before != "" && !httpx.IsUUID(before) {
		httpx.Fail(c, http.StatusBadRequest, "invalid cursor")
		return
	}
	page, err := h.service.List(c.Request.Context(), c.GetString("userID"), matchID, before, limit)
	if err != nil {
		h.fail(c, "messaging.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	var req SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), matchID, req.Body)
	if err != nil {
		h.fail(c, "messaging.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "messaging.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Clear(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "messaging.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
