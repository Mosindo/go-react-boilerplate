package matching

import (
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	req.TargetUserID = strings.ToLower(req.TargetUserID)
	if !httpx.IsUUID(req.TargetUserID) {
		httpx.Fail(c, notFoundProfile())
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) List(c *gin.Context) {
	limit, ok := httpx.QueryInt(c, "limit", DefaultLimit, 1, MaxLimit)
	if !ok {
		return
	}
	res, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}
