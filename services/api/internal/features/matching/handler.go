package matching

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Feed(c *gin.Context) {
	cards, err := h.service.Feed(c.Request.Context(), c.GetString("userID"), httpx.Limit(c, DefaultFeedSize, MaxFeedSize))
	if err != nil {
		httpx.Internal(c, "matching.feed", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, FeedResponse{Profiles: cards})
}

func (h *Handler) ReceivedLikes(c *gin.Context) {
	cards, err := h.service.ReceivedLikes(c.Request.Context(), c.GetString("userID"), httpx.Limit(c, DefaultFeedSize, MaxFeedSize))
	if err != nil {
		httpx.Internal(c, "matching.received", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, ReceivedLikesResponse{Profiles: cards})
}

func (h *Handler) Like(c *gin.Context) { h.swipe(c, KindLike) }
func (h *Handler) Pass(c *gin.Context) { h.swipe(c, KindPass) }

func (h *Handler) swipe(c *gin.Context, kind string) {
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), c.Param("id"), kind)
	switch {
	case err == nil:
		httpx.NoStore(c)
		c.JSON(http.StatusOK, res)
	case errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidKind), errors.Is(err, ErrSelf):
		httpx.BadRequest(c, "matching.swipe", err.Error(), err)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, "matching.swipe", http.StatusNotFound, "profile not found", nil)
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, "matching.swipe", http.StatusConflict, err.Error(), err)
	default:
		httpx.Internal(c, "matching.swipe", err)
	}
}

func (h *Handler) ListMatches(c *gin.Context) {
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.BadRequest(c, "matching.list", "invalid before cursor", err)
			return
		}
		before = &t
	}
	resp, err := h.service.ListMatches(c.Request.Context(), c.GetString("userID"), before, httpx.Limit(c, 20, MaxFeedSize))
	if err != nil {
		httpx.Internal(c, "matching.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, "matching.unmatch", http.StatusNotFound, "match not found", nil)
	default:
		httpx.Internal(c, "matching.unmatch", err)
	}
}
