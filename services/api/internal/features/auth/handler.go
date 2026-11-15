package auth

import (
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

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.Bind(c, "auth.register", &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.register", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusCreated, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMeResponse(user)})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.Bind(c, "auth.login", &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.login", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMeResponse(user)})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.Bind(c, "auth.refresh", &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.refresh", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMeResponse(user)})
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.Bind(c, "auth.logout", &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Fail(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.Bind(c, "auth.forgot", &req) {
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, "auth.forgot", err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if an account exists, a code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.Bind(c, "auth.reset", &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		httpx.Fail(c, "auth.reset", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "auth.me", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.Bind(c, "auth.change_password", &req) {
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword); err != nil {
		httpx.Fail(c, "auth.change_password", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.Bind(c, "auth.delete_account", &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		httpx.Fail(c, "auth.delete_account", err)
		return
	}
	c.Status(http.StatusNoContent)
}
