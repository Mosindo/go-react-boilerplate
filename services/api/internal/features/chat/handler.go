package chat

import (
	"net/http"
	"strconv"
	"time"

	perr "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func limitParam(c *gin.Context, def, max int) (int, bool) {
	v := c.Query("limit")
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > max {
		perr.Respond(c, perr.BadRequest("invalid_limit", "limit invalide."))
		return 0, false
	}
	return n, true
}

func (h *Handler) List(c *gin.Context) {
	limit, ok := limitParam(c, 30, 50)
	if !ok {
		return
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
	items, err := h.svc.List(c.Request.Context(), c.GetString("userID"), limit, before)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	var next *string
	if len(items) == limit {
		s := items[len(items)-1].ActivityAt.Format(time.RFC3339Nano)
		next = &s
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items, "nextCursor": next})
}

func (h *Handler) Messages(c *gin.Context) {
	limit, ok := limitParam(c, 30, 100)
	if !ok {
		return
	}
	var before *int64
	if v := c.Query("before"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			perr.Respond(c, perr.BadRequest("invalid_cursor", "Curseur invalide."))
			return
		}
		before = &n
	}
	items, err := h.svc.Messages(c.Request.Context(), c.Param("id"), c.GetString("userID"), limit, before)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	var next *int64
	if len(items) == limit {
		id := items[len(items)-1].ID
		next = &id
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items, "nextCursor": next})
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		perr.Respond(c, perr.BadRequest("invalid_request", "Requête invalide."))
		return
	}
	msg, err := h.svc.Send(c.Request.Context(), c.Param("id"), c.GetString("userID"), req.Body)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.svc.MarkRead(c.Request.Context(), c.Param("id"), c.GetString("userID")); err != nil {
		perr.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.svc.Hide(c.Request.Context(), c.Param("id"), c.GetString("userID")); err != nil {
		perr.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.svc.Unmatch(c.Request.Context(), c.Param("id"), c.GetString("userID")); err != nil {
		perr.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, sendLimit gin.HandlerFunc) {
	r.GET("/matches", requireUser, h.List)
	r.GET("/matches/:id/messages", requireUser, h.Messages)
	r.POST("/matches/:id/messages", requireUser, sendLimit, h.Send)
	r.POST("/matches/:id/read", requireUser, h.MarkRead)
	r.POST("/matches/:id/hide", requireUser, h.Hide)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
