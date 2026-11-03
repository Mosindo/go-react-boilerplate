package matching

import (
	"net/http"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// cursor parses the optional ?before= RFC3339 timestamp used for keyset paging.
func cursor(c *gin.Context) (time.Time, bool) {
	raw := c.Query("before")
	if raw == "" {
		return time.Now().Add(time.Minute), true
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		httpx.Fail(c, "matching.cursor", platformerrors.Invalid("before must be an RFC3339 timestamp"))
		return time.Time{}, false
	}
	return t, true
}

func (h *Handler) Discover(c *gin.Context) {
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 10, 20))
	if err != nil {
		httpx.Fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if _, ok := httpx.UUIDParamValue(req.TargetID); !ok {
		httpx.Fail(c, "matching.swipe", platformerrors.Invalid("invalid targetId"))
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Matches(c *gin.Context) {
	before, ok := cursor(c)
	if !ok {
		return
	}
	items, err := h.service.Matches(c.Request.Context(), httpx.UserID(c), before, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "matching.matches", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"matches": items})
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) LikesReceived(c *gin.Context) {
	before, ok := cursor(c)
	if !ok {
		return
	}
	items, err := h.service.LikesReceived(c.Request.Context(), httpx.UserID(c), before, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "matching.likes", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"likes": items})
}
