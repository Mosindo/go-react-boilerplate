package matches

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		httpx.Fail(c, "matches.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) List(c *gin.Context) {
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.BadRequest(c, "invalid 'before' cursor")
			return
		}
		before = &t
	}
	res, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "matches.list", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), c.Param("matchId")); err != nil {
		httpx.Fail(c, "matches.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipes *middleware.Limiter) {
	r.POST("/swipes", requireUser, swipes.ByUser(), h.Swipe)
	r.GET("/matches", requireUser, h.List)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
