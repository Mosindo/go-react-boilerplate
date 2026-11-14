package safety

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelf):
		httpx.BadRequest(c, "invalid_target", "action impossible sur votre propre compte")
	case errors.Is(err, ErrInvalidReason):
		httpx.BadRequest(c, "invalid_reason", "motif de signalement invalide")
	case errors.Is(err, ErrInvalidDetail):
		httpx.BadRequest(c, "invalid_details", "détails limités à 1000 caractères")
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "user_not_found", "membre introuvable")
	default:
		httpx.Internal(c, op, err)
	}
}

func targetID(c *gin.Context, raw string) (string, bool) {
	id := strings.ToLower(strings.TrimSpace(raw))
	if !httpx.IsUUID(id) {
		httpx.BadRequest(c, "invalid_id", "identifiant invalide")
		return "", false
	}
	return id, true
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	id, ok := targetID(c, req.UserID)
	if !ok {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Blocks(c *gin.Context) {
	list, err := h.service.Blocks(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 50, 100), httpx.Offset(c, 10_000))
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, BlocksResponse{Blocks: list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	id, ok := targetID(c, req.UserID)
	if !ok {
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), id, req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "received"})
}
