package matching

import (
	"errors"
	"net/http"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusConflict, "profile_incomplete", err.Error())
	case errors.Is(err, ErrSelfSwipe):
		httpx.Fail(c, http.StatusBadRequest, "invalid_target", err.Error())
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, http.StatusConflict, "already_swiped", err.Error())
	case errors.Is(err, ErrTargetUnavailable):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.FailInternal(c, operation, err)
	}
}

func cardsResponse(cards []profiles.Card) CardsResponse {
	out := make([]profiles.CardResponse, 0, len(cards))
	for _, card := range cards {
		out = append(out, profiles.ToCardResponse(card))
	}
	return CardsResponse{Profiles: out}
}

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.QueryLimit(c, 10, 25)
	if !ok {
		return
	}
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, cardsResponse(cards))
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, "matching.swipe", &req) {
		return
	}
	result, matched, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.TargetUserID, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	resp := SwipeResponse{Matched: result.Matched}
	if result.Matched {
		resp.MatchID = &result.MatchID
		resp.ConversationID = &result.ConversationID
		if matched != nil {
			summary := profiles.ToCardSummary(*matched)
			resp.MatchedUser = &summary
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) LikesReceived(c *gin.Context) {
	limit, ok := httpx.QueryLimit(c, 20, 50)
	if !ok {
		return
	}
	offset, ok := httpx.QueryOffset(c)
	if !ok {
		return
	}
	cards, err := h.service.LikesReceived(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "matching.likes_received", err)
		return
	}
	c.JSON(http.StatusOK, cardsResponse(cards))
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.PathUUID(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
