package notifications

import (
	"net/http"
	"strconv"
	"time"

	perr "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *gin.Context) {
	limit := 30
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			perr.Respond(c, perr.BadRequest("invalid_limit", "limit doit être entre 1 et 100."))
			return
		}
		limit = n
	}
	var before *time.Time
	if v := c.Query("cursor"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			perr.Respond(c, perr.BadRequest("invalid_cursor", "Curseur invalide."))
			return
		}
		before = &t
	}
	res, err := h.svc.List(c.Request.Context(), c.GetString("userID"), limit, before)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) MarkRead(c *gin.Context) {
	ok, err := h.svc.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		perr.Respond(c, perr.NotFound("Notification introuvable."))
		return
	}
	if !ok {
		perr.Respond(c, perr.NotFound("Notification introuvable."))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.svc.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		perr.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
