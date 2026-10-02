package auth

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func bind(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		httpx.BadRequest(c, "invalid request")
		return false
	}
	return true
}

func writeAuth(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         UserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !bind(c, &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		writeAuth(c, http.StatusCreated, tokens, user)
		logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
	case errors.Is(err, ErrInvalidEmail):
		httpx.Fail(c, http.StatusBadRequest, "invalid_email", "invalid email address")
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", "password must be between 8 and 72 characters")
	case errors.Is(err, ErrEmailExists):
		httpx.Fail(c, http.StatusConflict, "email_exists", "email already exists")
	default:
		httpx.Internal(c, "auth.register", err)
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !bind(c, &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		writeAuth(c, http.StatusOK, tokens, user)
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	default:
		httpx.Internal(c, "auth.login", err)
	}
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !bind(c, &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		writeAuth(c, http.StatusOK, tokens, user)
	case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "invalid refresh token")
	default:
		httpx.Internal(c, "auth.refresh", err)
	}
}

func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if !bind(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Internal(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, UserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt})
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "user not found")
	default:
		httpx.Internal(c, "auth.me", err)
	}
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !bind(c, &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		// Do not leak whether the account exists: log, but answer the same way.
		logger.LogHandlerError(c, "auth.forgot_password", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if the account exists, a code was sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !bind(c, &req) {
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", "password must be between 8 and 72 characters")
	case errors.Is(err, ErrInvalidResetCode):
		httpx.Fail(c, http.StatusBadRequest, "invalid_code", "invalid or expired code")
	default:
		httpx.Internal(c, "auth.reset_password", err)
	}
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !bind(c, &req) {
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", "password must be between 8 and 72 characters")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, "wrong_password", "current password is incorrect")
	default:
		httpx.Internal(c, "auth.change_password", err)
	}
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !bind(c, &req) {
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password)
	switch {
	case err == nil:
		logger.LogHandlerEvent(c, "auth.delete_account", http.StatusNoContent, nil)
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, "wrong_password", "password is incorrect")
	default:
		httpx.Internal(c, "auth.delete_account", err)
	}
}
