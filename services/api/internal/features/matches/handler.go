package matches

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Swipe(c *gin.Context) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.BadRequest(c, "userId must be a valid id")
		return
	}
	if req.Action != ActionLike && req.Action != ActionPass {
		httpx.BadRequest(c, "action must be \"like\" or \"pass\"")
		return
	}
	viewer := c.GetString("userID")
	if req.UserID == viewer {
		httpx.BadRequest(c, "you cannot swipe on yourself")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), viewer, req.UserID, req.Action)
	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyActed):
			httpx.Error(c, http.StatusConflict, err.Error())
		case errors.Is(err, ErrNotEligible):
			httpx.Error(c, http.StatusNotFound, err.Error())
		default:
			logger.LogHandlerError(c, "matches.swipe", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, res)
}

func (h *Handler) List(c *gin.Context) {
	limit := DefaultLimit
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			httpx.BadRequest(c, "limit must be between 1 and 50")
			return
		}
		limit = n
	}
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.BadRequest(c, "before must be an RFC 3339 timestamp")
			return
		}
		before = &t
	}
	resp, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		logger.LogHandlerError(c, "matches.list", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "match not found")
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "match not found")
			return
		}
		logger.LogHandlerError(c, "matches.unmatch", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
