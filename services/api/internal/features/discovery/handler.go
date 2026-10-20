package discovery

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.Limit(c, 10, 30)
	if !ok {
		return
	}
	list, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		httpx.Respond(c, "discovery.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.TargetUserID, req.Action)
	if err != nil {
		httpx.Respond(c, "discovery.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListMatches(c *gin.Context) {
	limit, ok := httpx.Limit(c, 20, 50)
	if !ok {
		return
	}
	cursor, ok := httpx.DecodeTimeCursor(c.Query("cursor"))
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid cursor")
		return
	}
	res, err := h.service.ListMatches(c.Request.Context(), httpx.UserID(c), cursor, limit)
	if err != nil {
		httpx.Respond(c, "discovery.matches", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Respond(c, "discovery.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
