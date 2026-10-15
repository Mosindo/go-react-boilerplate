package matching

import (
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Discover(c *gin.Context) {
	limit, _, ok := httpx.Page(c, defaultDiscoverLimit, maxDiscoverLimit)
	if !ok {
		return
	}
	var exclude []string
	if raw := strings.TrimSpace(c.Query("exclude")); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			id = strings.ToLower(strings.TrimSpace(id))
			if !httpx.IsUUID(id) || len(exclude) >= maxExclude {
				httpx.Fail(c, "matching.discover", ErrInvalidExclude)
				return
			}
			exclude = append(exclude, id)
		}
	}
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), exclude, limit)
	if err != nil {
		httpx.Fail(c, "matching.discover", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: cards})
}

func (h *Handler) Profile(c *gin.Context) {
	id := strings.ToLower(c.Param("userId"))
	if !httpx.IsUUID(id) {
		httpx.Fail(c, "matching.profile", ErrProfileNotFound)
		return
	}
	card, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "matching.profile", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, card)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.Bind(c, &req) {
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.UserID, req.Action)
	if err != nil {
		httpx.Fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id := strings.ToLower(c.Param("userId"))
	if !httpx.IsUUID(id) {
		httpx.Fail(c, "matching.unmatch", ErrNotMatched)
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
