package safety

import (
	"errors"
	"net/http"
	"regexp"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil || !uuidPattern.MatchString(req.UserID) {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id := c.Param("userId")
	if !uuidPattern.MatchString(id) {
		httpx.Error(c, http.StatusNotFound, "user not found")
		return
	}
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.ListBlocks(c.Request.Context(), c.GetString("userID"), httpx.Limit(c.Query("limit"), 50, 100), httpx.Offset(c.Query("offset")))
	if err != nil {
		h.fail(c, "safety.list", err)
		return
	}
	payload := make([]BlockedUserResponse, 0, len(items))
	for _, b := range items {
		payload = append(payload, BlockedUserResponse{UserID: b.UserID, FirstName: b.FirstName, BlockedAt: b.BlockedAt})
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, BlocksResponse{Blocks: payload})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil || !uuidPattern.MatchString(req.UserID) {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusNotFound, "user not found")
	case errors.Is(err, ErrSelfAction), errors.Is(err, ErrInvalidReason):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}
