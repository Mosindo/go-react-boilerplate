package matches

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrInvalidAction), errors.Is(err, ErrSelfSwipe), errors.Is(err, ErrBadCursor):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, ErrNotEligible):
		httpx.Fail(c, http.StatusConflict, "not_available", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "matches.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) List(c *gin.Context) {
	limit, ok := httpx.QueryInt(c, "limit", 30, 1, 50)
	if !ok {
		return
	}
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), limit)
	if err != nil {
		h.fail(c, "matches.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "matches.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
