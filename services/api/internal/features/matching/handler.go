package matching

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

func (h *Handler) Deck(c *gin.Context) {
	deck, err := h.service.Deck(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, defaultDeckSize, maxDeckSize))
	if err != nil {
		httpx.Fail(c, "matching.deck", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, deck)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	result, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *Handler) ListMatches(c *gin.Context) {
	cursor, ok := httpx.DecodeCursor(c)
	if !ok {
		return
	}
	page, err := h.service.ListMatches(c.Request.Context(), httpx.UserID(c), cursor, httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "matching.list_matches", err)
		return
	}
	c.JSON(http.StatusOK, page)
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

func (h *Handler) LikesReceived(c *gin.Context) {
	likes, err := h.service.LikesReceived(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "matching.likes_received", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"profiles": likes})
}
