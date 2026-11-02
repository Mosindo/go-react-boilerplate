package discovery

import (
	"net/http"
	"strconv"

	perr "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Deck(c *gin.Context) {
	limit := DefaultDeck
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > MaxDeck {
			perr.Respond(c, perr.BadRequest("invalid_limit", "limit doit être entre 1 et 20."))
			return
		}
		limit = n
	}
	cards, err := h.svc.Deck(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": cards})
}

func (h *Handler) swipe(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := h.svc.Swipe(c.Request.Context(), c.GetString("userID"), c.Param("id"), action)
		if err != nil {
			perr.Respond(c, err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

func (h *Handler) Profile(c *gin.Context) {
	card, err := h.svc.Profile(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, card)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, actionLimit gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Deck)
	r.POST("/discover/:id/like", requireUser, actionLimit, h.swipe("like"))
	r.POST("/discover/:id/pass", requireUser, actionLimit, h.swipe("pass"))
	r.GET("/users/:id", requireUser, h.Profile)
}
