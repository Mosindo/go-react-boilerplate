package auth

import (
	"net/http"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service      *Service
	loginLimiter *middleware.RateLimiter // per email, slows credential stuffing
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service, loginLimiter: middleware.NewRateLimiter(10, 5)}
}

func (h *Handler) respondAuth(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMe(user)})
}

func (h *Handler) Register(c *gin.Context) {
	var req Credentials
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.register", err)
		return
	}
	h.respondAuth(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req Credentials
	if !httpx.BindJSON(c, &req) {
		return
	}
	if ok, _ := h.loginLimiter.Allow(strings.ToLower(strings.TrimSpace(req.Email))); !ok {
		c.Header("Retry-After", "60")
		httpx.Fail(c, "auth.login", &platformerrors.Error{Kind: platformerrors.KindRateLimited, Code: "rate_limited", Message: "Too many attempts, try again in a minute"})
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.login", err)
		return
	}
	h.respondAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.refresh", err)
		return
	}
	h.respondAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Fail(c, "auth.logout", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, "auth.forgot", err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		httpx.Fail(c, "auth.reset", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMe(user))
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword); err != nil {
		httpx.Fail(c, "auth.change_password", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		httpx.Fail(c, "auth.delete_account", err)
		return
	}
	httpx.NoContent(c)
}
