package discovery

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Deck(c *gin.Context) {
	resp, err := h.service.Deck(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, DefaultDeckSize, MaxDeckSize))
	if err != nil {
		httpx.Fail(c, "discovery.deck", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) LikesReceived(c *gin.Context) {
	resp, err := h.service.LikesReceived(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "discovery.likes", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.BadRequest(c, "validation", "invalid userId")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.UserID, req.Action)
	if err != nil {
		httpx.Fail(c, "discovery.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
