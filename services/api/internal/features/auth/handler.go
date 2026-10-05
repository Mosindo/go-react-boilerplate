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

func authResponse(tokens Tokens, user User) AuthResponse {
	return AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         meResponse(user),
	}
}

func meResponse(user User) MeResponse {
	return MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, "auth.register", &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			httpx.Fail(c, "auth.register", http.StatusConflict, "email_exists", "email already exists", err)
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, "auth.register", http.StatusBadRequest, "weak_password", err.Error(), err)
		case errors.Is(err, ErrInvalidEmail):
			httpx.Fail(c, "auth.register", http.StatusBadRequest, "invalid_email", err.Error(), err)
		default:
			httpx.Internal(c, "auth.register", err)
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, authResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, "auth.login", &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, "auth.login", http.StatusUnauthorized, "invalid_credentials", "invalid credentials", err)
			return
		}
		httpx.Internal(c, "auth.login", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, "auth.refresh", &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, "auth.refresh", http.StatusUnauthorized, "invalid_refresh_token", "invalid refresh token", err)
			return
		}
		httpx.Internal(c, "auth.refresh", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, "auth.logout", &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.Fail(c, "auth.logout", http.StatusBadRequest, "invalid_refresh_token", "invalid refresh token", err)
			return
		}
		httpx.Internal(c, "auth.logout", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, "auth.me", http.StatusUnauthorized, "unauthorized", "user not found", err)
			return
		}
		httpx.Internal(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, meResponse(user))
}

// ForgotPassword answers 202 whether or not the address is registered.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, "auth.forgot", &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, "auth.reset", &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidResetToken):
			httpx.Fail(c, "auth.reset", http.StatusBadRequest, "invalid_reset_token", "invalid or expired reset token", err)
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, "auth.reset", http.StatusBadRequest, "weak_password", err.Error(), err)
		default:
			httpx.Internal(c, "auth.reset", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.BindJSON(c, "auth.change_password", &req) {
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, "auth.change_password", http.StatusForbidden, "invalid_credentials", "current password is incorrect", err)
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, "auth.change_password", http.StatusBadRequest, "weak_password", err.Error(), err)
		default:
			httpx.Internal(c, "auth.change_password", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, "auth.delete_account", &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, "auth.delete_account", http.StatusForbidden, "invalid_credentials", "password is incorrect", err)
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, "auth.delete_account", http.StatusNotFound, "not_found", "user not found", err)
		default:
			httpx.Internal(c, "auth.delete_account", err)
		}
		return
	}
	logger.LogHandlerEvent(c, "auth.delete_account.success", http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}
