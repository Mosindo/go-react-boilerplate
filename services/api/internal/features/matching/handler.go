package matching

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
	resp, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 10, 30))
	if err != nil {
		httpx.Fail(c, "matching.discover", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.Bind(c, "matching.swipe", &req) {
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ReceivedLikes(c *gin.Context) {
	cursor, err := httpx.DecodeCursor(c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "matching.likes", err)
		return
	}
	resp, err := h.service.ReceivedLikes(c.Request.Context(), httpx.UserID(c), cursor, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "matching.likes", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		httpx.Fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
