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

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, "safety.block", &req) {
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.UserID))
	if !httpx.IsUUID(target) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "Identifiant invalide.")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), target); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	target, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), target); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	blocks, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.FailInternal(c, "safety.list_blocks", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, BlocksResponse{Blocks: blocks})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, "safety.report", &req) {
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.UserID))
	if !httpx.IsUUID(target) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "Identifiant invalide.")
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), target, req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "received"})
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrCannotTargetYou):
		httpx.Fail(c, http.StatusBadRequest, "cannot_target_self", "Action impossible sur ton propre compte.")
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "user_not_found", "Membre introuvable.")
	case errors.Is(err, ErrBlockNotFound):
		httpx.Fail(c, http.StatusNotFound, "block_not_found", "Ce membre n'est pas bloqué.")
	case errors.Is(err, ErrInvalidReason):
		httpx.Fail(c, http.StatusBadRequest, "invalid_reason", "Motif de signalement invalide.")
	case errors.Is(err, ErrDetailsTooLong):
		httpx.Fail(c, http.StatusBadRequest, "details_too_long", "Détails trop longs (1000 caractères max).")
	default:
		httpx.FailInternal(c, op, err)
	}
}
