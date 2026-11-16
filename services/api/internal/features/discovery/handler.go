package discovery

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Deck(c *gin.Context) {
	resp, err := h.service.Deck(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), DefaultLimit, MaxLimit))
	if err != nil {
		if errors.Is(err, ErrViewerIncomplete) {
			httpx.Fail(c, http.StatusConflict, "profile_incomplete", err.Error())
			return
		}
		logger.LogHandlerError(c, "discovery.deck", http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not load profiles")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}
