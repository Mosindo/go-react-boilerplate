package matching

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.Limit(c, DefaultFeedSize, MaxFeedSize)
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "invalid limit")
		return
	}
	cards, err := h.service.Feed(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		logger.LogHandlerError(c, "matching.feed", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": cards})
}

func (h *Handler) swipe(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParamUUID(c, "id")
		if !ok {
			return
		}
		res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), id, action)
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.Error(c, http.StatusNotFound, "profile not found")
		case errors.Is(err, ErrAlreadySwiped):
			httpx.Error(c, http.StatusConflict, "already swiped")
		case errors.Is(err, ErrProfileMissing):
			httpx.Error(c, http.StatusConflict, "create your profile first")
		case err != nil:
			logger.LogHandlerError(c, "matching.swipe", http.StatusInternalServerError, err)
			httpx.Internal(c)
		default:
			c.JSON(http.StatusOK, res)
		}
	}
}

func (h *Handler) ListMatches(c *gin.Context) {
	limit, ok := httpx.Limit(c, 30, 50)
	cur, okCur := httpx.DecodeCursor(c)
	if !ok || !okCur {
		httpx.Error(c, http.StatusBadRequest, "invalid pagination")
		return
	}
	items, err := h.service.ListMatches(c.Request.Context(), c.GetString("userID"), limit, cur)
	if err != nil {
		logger.LogHandlerError(c, "matching.list", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	next := ""
	if len(items) == limit {
		last := items[len(items)-1]
		next = httpx.EncodeCursor(last.CreatedAt, last.ID)
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items, "nextCursor": next})
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "match not found")
	case err != nil:
		logger.LogHandlerError(c, "matching.unmatch", http.StatusInternalServerError, err)
		httpx.Internal(c)
	default:
		c.Status(http.StatusNoContent)
	}
}
