package matches

import (
	"errors"
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

func (h *Handler) List(c *gin.Context) {
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), httpx.Limit(c, DefaultMatchesLimit, MaxMatchesLimit))
	if err != nil {
		h.fail(c, "matches.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, MatchesResponse{Matches: page.Matches, NextCursor: page.NextCursor})
}

func (h *Handler) Get(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	match, err := h.service.Get(c.Request.Context(), httpx.UserID(c), matchID)
	if err != nil {
		h.fail(c, "matches.get", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, match)
}

func (h *Handler) Messages(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	msgs, next, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), matchID, c.Query("cursor"), httpx.Limit(c, DefaultMessagesLimit, MaxMessagesLimit))
	if err != nil {
		h.fail(c, "matches.messages", err)
		return
	}
	out := make([]MessageResponse, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, ToMessageResponse(m))
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, MessagesResponse{Messages: out, NextCursor: next})
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "message body is required")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), matchID, req.Body)
	if err != nil {
		h.fail(c, "matches.send", err)
		return
	}
	c.JSON(http.StatusCreated, ToMessageResponse(msg))
}

func (h *Handler) MarkRead(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "matches.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "matches.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		// Same response whether the match does not exist or belongs to others.
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrEmptyMessage), errors.Is(err, ErrMessageLong):
		httpx.BadRequest(c, "invalid_message", err.Error())
	case errors.Is(err, ErrInvalidCursor):
		httpx.BadRequest(c, "invalid_cursor", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
