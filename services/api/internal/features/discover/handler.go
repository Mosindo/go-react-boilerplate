package discover

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Next(c *gin.Context) {
	limit := DefaultLimit
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			httpx.BadRequest(c, "limit must be between 1 and 20")
			return
		}
		limit = n
	}
	items, err := h.service.Next(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		if errors.Is(err, ErrIncompleteProfile) {
			httpx.Error(c, http.StatusConflict, err.Error())
			return
		}
		logger.LogHandlerError(c, "discover.next", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, Response{Profiles: items})
}
