package safety

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	if v, ok := httpx.AsValidation(err); ok {
		httpx.Error(c, http.StatusBadRequest, v.Message)
		return
	}
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusNotFound, "user not found")
	case errors.Is(err, ErrReportNotFound):
		httpx.Error(c, http.StatusNotFound, "report not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	list, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), 50, 100), httpx.Offset(c.Query("offset")))
	if err != nil {
		h.fail(c, "safety.list_blocks", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, BlocksResponse{Blocks: list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	rep, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": rep.ID, "status": rep.Status})
}

func (h *Handler) AdminListReports(c *gin.Context) {
	list, err := h.service.ListReports(c.Request.Context(), c.Query("status"), httpx.Limit(c.Query("limit"), 20, 100), httpx.Offset(c.Query("offset")))
	if err != nil {
		h.fail(c, "safety.admin_list", err)
		return
	}
	c.JSON(http.StatusOK, ReportsResponse{Reports: list})
}

func (h *Handler) AdminResolveReport(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "reportId")
	if !ok {
		return
	}
	var req ResolveReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	rep, err := h.service.ResolveReport(c.Request.Context(), id, req)
	if err != nil {
		h.fail(c, "safety.admin_resolve", err)
		return
	}
	c.JSON(http.StatusOK, rep)
}
