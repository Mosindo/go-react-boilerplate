package moderation

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "utilisateur introuvable")
	case errors.Is(err, ErrSelf):
		httpx.BadRequest(c, "action impossible sur votre propre compte")
	case errors.Is(err, ErrInvalidReason):
		httpx.BadRequest(c, "motif de signalement invalide")
	case errors.Is(err, ErrDetailsLong):
		httpx.BadRequest(c, "détails trop longs (1000 caractères maximum)")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId requis")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		fail(c, "moderation.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), c.Param("userId")); err != nil {
		fail(c, "moderation.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.ListBlocked(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		fail(c, "moderation.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId et motif requis")
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req); err != nil {
		fail(c, "moderation.report", err)
		return
	}
	c.Status(http.StatusAccepted)
}
