package chat

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "conversation introuvable")
	case errors.Is(err, ErrEmptyMessage):
		httpx.BadRequest(c, "le message est vide")
	case errors.Is(err, ErrTooLong):
		httpx.BadRequest(c, "le message est trop long (2000 caractères maximum)")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.Conversations(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, DefaultPage, MaxPage), httpx.Offset(c))
	if err != nil {
		fail(c, "chat.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": items})
}

func (h *Handler) Messages(c *gin.Context) {
	before, ok, err := httpx.Cursor(c, "before")
	if err != nil {
		httpx.BadRequest(c, "curseur invalide")
		return
	}
	var cursor *time.Time
	if ok {
		cursor = &before
	}
	items, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), c.Param("id"), httpx.Limit(c, DefaultPage, MaxPage), cursor)
	if err != nil {
		fail(c, "chat.messages", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": items})
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "le message est vide")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), c.Param("id"), req.Body)
	if err != nil {
		fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"read": n})
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	send := middleware.NewLimiter(60, time.Minute).ByUser()
	r.GET("/conversations", requireUser, h.List)
	r.GET("/conversations/:id/messages", requireUser, h.Messages)
	r.POST("/conversations/:id/messages", requireUser, send, h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
}
