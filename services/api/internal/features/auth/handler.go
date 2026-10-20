package auth

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func writeSession(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: user})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Respond(c, "auth.register", err)
		return
	}
	writeSession(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Respond(c, "auth.login", err)
		return
	}
	writeSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Respond(c, "auth.refresh", err)
		return
	}
	writeSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Respond(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, user)
}

func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req ResetRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		httpx.Respond(c, "auth.reset.request", err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{})
}

func (h *Handler) ConfirmPasswordReset(c *gin.Context) {
	var req ResetConfirmRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ConfirmPasswordReset(c.Request.Context(), req.Token, req.Password); err != nil {
		httpx.Respond(c, "auth.reset.confirm", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		httpx.Respond(c, "auth.delete", err)
		return
	}
	logger.LogHandlerEvent(c, "auth.delete.success", http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}
