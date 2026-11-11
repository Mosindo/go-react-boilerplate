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
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), strings.ToLower(req.UserID)); err != nil {
		fail(c, err)
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
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	list, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, BlocksResponse{Blocks: list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	id, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, ReportResponse{ID: id})
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrSelf):
		httpx.BadRequest(c, "self_target", "Action impossible sur ton propre compte.")
	case errors.Is(err, ErrDetailsTooLong):
		httpx.BadRequest(c, "details_too_long", "Description trop longue (1000 caractères maximum).")
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "user_not_found", "Utilisateur introuvable.", nil)
	default:
		httpx.Internal(c, err)
	}
}
