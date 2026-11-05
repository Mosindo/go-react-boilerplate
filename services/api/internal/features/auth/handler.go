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

func toUserResponse(u User) UserResponse {
	return UserResponse{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt}
}

func (h *Handler) respondSession(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toUserResponse(user),
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req Credentials
	if !httpx.Bind(c, &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrPasswordTooLong) {
			httpx.BadRequest(c, "password is too long")
			return
		}
		if errors.Is(err, ErrInvalidEmail) {
			httpx.BadRequest(c, "invalid email address")
			return
		}
		if errors.Is(err, ErrEmailExists) {
			httpx.Error(c, http.StatusConflict, "email already exists")
			return
		}
		logger.LogHandlerError(c, "auth.register", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	h.respondSession(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.Bind(c, &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Error(c, http.StatusUnauthorized, "invalid credentials")
			return
		}
		logger.LogHandlerError(c, "auth.login", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	h.respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.Bind(c, &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.Error(c, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		logger.LogHandlerError(c, "auth.refresh", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	h.respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.BadRequest(c, "invalid refresh token")
			return
		}
		logger.LogHandlerError(c, "auth.logout", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Error(c, http.StatusUnauthorized, "user not found")
			return
		}
		logger.LogHandlerError(c, "auth.me", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toUserResponse(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		logger.LogHandlerError(c, "auth.forgot", http.StatusInternalServerError, err)
	}
	// Always 202 so the endpoint cannot be used to enumerate accounts.
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, ErrPasswordTooLong) {
			httpx.BadRequest(c, "password is too long")
			return
		}
		if errors.Is(err, ErrInvalidResetToken) {
			httpx.BadRequest(c, "invalid or expired reset token")
			return
		}
		logger.LogHandlerError(c, "auth.reset", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrPasswordTooLong) {
			httpx.BadRequest(c, "password is too long")
			return
		}
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Error(c, http.StatusForbidden, "current password is incorrect")
			return
		}
		logger.LogHandlerError(c, "auth.change_password", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Error(c, http.StatusForbidden, "password is incorrect")
			return
		}
		logger.LogHandlerError(c, "auth.delete_account", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
