package auth

import (
	"errors"
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

func bindFailed(c *gin.Context, op string, err error) {
	logger.LogHandlerError(c, op, http.StatusBadRequest, err)
	httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request")
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, code, msg := http.StatusInternalServerError, "internal", "something went wrong"
	switch {
	case errors.Is(err, ErrEmailExists):
		status, code, msg = http.StatusConflict, "email_exists", "an account already exists for this email"
	case errors.Is(err, ErrInvalidEmail):
		status, code, msg = http.StatusBadRequest, "invalid_email", err.Error()
	case errors.Is(err, ErrWeakPassword):
		status, code, msg = http.StatusBadRequest, "weak_password", err.Error()
	case errors.Is(err, ErrInvalidCredentials):
		status, code, msg = http.StatusUnauthorized, "invalid_credentials", "invalid email or password"
	case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
		status, code, msg = http.StatusUnauthorized, "unauthorized", "session expired"
	case errors.Is(err, ErrInvalidResetCode):
		status, code, msg = http.StatusBadRequest, "invalid_reset_code", err.Error()
	}
	logger.LogHandlerError(c, op, status, err)
	httpx.Fail(c, status, code, msg)
}

func (h *Handler) respondSession(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMeResponse(user)})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bindFailed(c, "auth.register.bind", err)
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.fail(c, "auth.register", err)
		return
	}
	h.respondSession(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bindFailed(c, "auth.login.bind", err)
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.fail(c, "auth.login", err)
		return
	}
	h.respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bindFailed(c, "auth.refresh.bind", err)
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.fail(c, "auth.refresh", err)
		return
	}
	h.respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bindFailed(c, "auth.logout.bind", err)
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.fail(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bindFailed(c, "auth.forgot.bind", err)
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		h.fail(c, "auth.forgot", err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if an account exists, a code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bindFailed(c, "auth.reset.bind", err)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		h.fail(c, "auth.reset", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bindFailed(c, "auth.change_password.bind", err)
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), req.CurrentPassword, req.NewPassword); err != nil {
		h.fail(c, "auth.change_password", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		bindFailed(c, "auth.delete.bind", err)
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		h.fail(c, "auth.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}
