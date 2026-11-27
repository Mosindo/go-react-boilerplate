package safety

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, err error) {
	if httpx.WriteValidation(c, err) {
		return
	}
	if errors.Is(err, ErrUserNotFound) {
		httpx.NotFound(c, "Utilisateur introuvable.")
		return
	}
	httpx.Internal(c, err)
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), req.UserID); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), userID); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	list, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	resp := BlocksResponse{Blocks: make([]BlockedUserResponse, 0, len(list))}
	for _, b := range list {
		resp.Blocks = append(resp.Blocks, BlockedUserResponse(b))
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	resp, err := h.service.Report(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}
