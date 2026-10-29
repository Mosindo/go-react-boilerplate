package discovery

import (
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

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.Limit(c, DefaultBatch, MaxBatch)
	if !ok {
		return
	}
	items, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		httpx.Fail(c, "discovery.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"profiles": items})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		httpx.Fail(c, "discovery.swipe", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		httpx.Fail(c, "discovery.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UnmatchConversation(c *gin.Context) {
	if err := h.service.UnmatchConversation(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		httpx.Fail(c, "discovery.unmatch_conversation", err)
		return
	}
	c.Status(http.StatusNoContent)
}
