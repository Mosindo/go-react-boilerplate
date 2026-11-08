package auth

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) respondTokens(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: user})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.register", err)
		return
	}
	h.respondTokens(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.login", err)
		return
	}
	h.respondTokens(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.refresh", err)
		return
	}
	h.respondTokens(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Fail(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "auth.me", err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, "auth.forgot", err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if the account exists, a recovery code was sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Code, req.NewPassword); err != nil {
		httpx.Fail(c, "auth.reset", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), req.CurrentPassword, req.NewPassword); err != nil {
		httpx.Fail(c, "auth.change_password", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		httpx.Fail(c, "auth.delete_account", err)
		return
	}
	c.Status(http.StatusNoContent)
}
