package discovery

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

func (h *Handler) Discover(c *gin.Context) {
	items, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), 10, 20))
	if err != nil {
		httpx.Respond(c, "discovery.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, httpx.Items(items))
}

func (h *Handler) Profile(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	profile, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Respond(c, "discovery.profile", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.TargetUserID) {
		httpx.Fail(c, http.StatusBadRequest, "validation_error", "targetUserId must be a UUID")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Respond(c, "discovery.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Matches(c *gin.Context) {
	items, err := h.service.Matches(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), 20, 50), httpx.Offset(c.Query("offset")))
	if err != nil {
		httpx.Respond(c, "discovery.matches", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, httpx.Items(items))
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Respond(c, "discovery.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
