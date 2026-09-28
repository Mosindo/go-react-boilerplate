package discovery

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	limit, err := httpx.ParseLimit(c, DefaultLimit, MaxLimit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	items, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, DiscoverResponse{Items: items})
}

func (h *Handler) Profile(c *gin.Context) {
	cand, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, cand)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UndoNotImplemented documents that there is no undo (contract).
func (h *Handler) UndoNotImplemented(c *gin.Context) {
	httpx.Abort(c, http.StatusNotImplemented, httpx.CodeInvalidRequest, "undo is not implemented")
}
