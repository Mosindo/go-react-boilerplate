package moderation

import (
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RequireModerator checks the role in the database on every request, so a
// revoked role takes effect immediately.
func (h *Handler) RequireModerator(c *gin.Context) {
	ok, err := h.service.IsModerator(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "moderation.role", err)
		return
	}
	if !ok {
		httpx.Fail(c, "moderation.role", ErrNotModerator)
		return
	}
	c.Next()
}

func (h *Handler) ListReports(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	resp, err := h.service.ListReports(c.Request.Context(), httpx.UserID(c), c.Query("status"), httpx.QueryLimit(c, 20, 50), offset)
	if err != nil {
		httpx.Fail(c, "moderation.reports", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ResolveReport(c *gin.Context) {
	reportID, ok := httpx.UUIDParam(c, "reportId")
	if !ok {
		return
	}
	var req ResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "moderation.resolve.bind", err)
		return
	}
	if err := h.service.ResolveReport(c.Request.Context(), httpx.UserID(c), reportID, req.Status); err != nil {
		httpx.Fail(c, "moderation.resolve", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Suspend(c *gin.Context)   { h.setSuspended(c, true) }
func (h *Handler) Unsuspend(c *gin.Context) { h.setSuspended(c, false) }

func (h *Handler) setSuspended(c *gin.Context, suspended bool) {
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.SetSuspended(c.Request.Context(), httpx.UserID(c), userID, suspended); err != nil {
		httpx.Fail(c, "moderation.suspend", err)
		return
	}
	c.Status(http.StatusNoContent)
}
