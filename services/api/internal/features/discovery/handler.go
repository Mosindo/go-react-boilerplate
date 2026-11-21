package discovery

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Feed(c *gin.Context) {
	limit := httpx.ParseLimit(c.Query("limit"), DefaultFeedSize, MaxFeedSize)
	exclude := []string{}
	if raw := strings.TrimSpace(c.Query("exclude")); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			id = strings.ToLower(strings.TrimSpace(id))
			if !httpx.IsUUID(id) {
				httpx.Error(c, http.StatusBadRequest, "invalid_exclude", "exclude must be a comma separated list of user ids")
				return
			}
			exclude = append(exclude, id)
		}
	}

	items, err := h.service.Feed(c.Request.Context(), httpx.UserID(c), limit, exclude)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrProfileIncomplete) {
			status = http.StatusConflict
			httpx.Error(c, status, "profile_incomplete", err.Error())
		} else {
			httpx.Internal(c, "could not load profiles")
		}
		logger.LogHandlerError(c, "discovery.feed", status, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, FeedResponse{Profiles: items})
}
