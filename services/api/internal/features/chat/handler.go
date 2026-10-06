package chat

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

func (h *Handler) Matches(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), defaultMatchesLimit, maxMatchesLimit)
	offset := httpx.Offset(c.Query("offset"))
	items, unread, err := h.service.ListMatches(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "chat.matches", err)
		return
	}
	payload := make([]MatchResponse, 0, len(items))
	for _, it := range items {
		r := MatchResponse{
			ID:          it.ID,
			User:        PeerResponse{UserID: it.Peer.UserID, FirstName: it.Peer.FirstName},
			CreatedAt:   it.CreatedAt,
			UnreadCount: it.UnreadCount,
		}
		if it.Peer.PrimaryPhoto != "" {
			r.User.PrimaryPhoto = "/photos/" + it.Peer.PrimaryPhoto + "/content"
		}
		if it.LastMessage != nil {
			lm := ToMessageResponse(*it.LastMessage)
			r.LastMessage = &lm
		}
		payload = append(payload, r)
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, MatchesResponse{Matches: payload, UnreadTotal: unread})
}

func (h *Handler) Messages(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	before := c.Query("before")
	if before != "" && !uuidPattern.MatchString(before) {
		httpx.Error(c, http.StatusBadRequest, "invalid cursor")
		return
	}
	msgs, hasMore, peerRead, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), matchID, before, httpx.Limit(c.Query("limit"), defaultMessagesLimit, maxMessagesLimit))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	payload := make([]MessageResponse, 0, len(msgs))
	for _, m := range msgs {
		payload = append(payload, ToMessageResponse(m))
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, MessagesResponse{Messages: payload, HasMore: hasMore, PeerLastReadAt: peerRead})
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), matchID, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, ToMessageResponse(msg))
}

func (h *Handler) MarkRead(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "chat.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func matchParam(c *gin.Context) (string, bool) {
	id := c.Param("matchId")
	if !uuidPattern.MatchString(id) {
		httpx.Error(c, http.StatusNotFound, "match not found")
		return "", false
	}
	return id, true
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrMatchNotFound):
		httpx.Error(c, http.StatusNotFound, "match not found")
	case errors.Is(err, ErrEmptyMessage), errors.Is(err, ErrMessageTooBig):
		httpx.Error(c, http.StatusUnprocessableEntity, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}
