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

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "auth.register.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrEmailExists):
			status = http.StatusConflict
			c.JSON(status, gin.H{"error": "email already exists"})
		default:
			c.JSON(status, gin.H{"error": "could not create user"})
		}
		logger.LogHandlerError(c, "auth.register", status, err)
		return
	}

	status := http.StatusCreated
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: MeResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
	logger.LogHandlerEvent(c, "auth.register.success", status, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "auth.login.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			status = http.StatusUnauthorized
			c.JSON(status, gin.H{"error": "invalid credentials"})
		default:
			c.JSON(status, gin.H{"error": "could not login"})
		}
		logger.LogHandlerError(c, "auth.login", status, err)
		return
	}

	status := http.StatusOK
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: MeResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
	logger.LogHandlerEvent(c, "auth.login.success", status, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "auth.refresh.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrInvalidRefreshToken):
			status = http.StatusUnauthorized
			c.JSON(status, gin.H{"error": "invalid refresh token"})
		case errors.Is(err, ErrUserNotFound):
			status = http.StatusUnauthorized
			c.JSON(status, gin.H{"error": "user not found"})
		default:
			c.JSON(status, gin.H{"error": "could not refresh session"})
		}
		logger.LogHandlerError(c, "auth.refresh", status, err)
		return
	}

	status := http.StatusOK
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: MeResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
	logger.LogHandlerEvent(c, "auth.refresh.success", status, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "auth.logout.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrInvalidRefreshToken):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "invalid refresh token"})
		default:
			c.JSON(status, gin.H{"error": "could not logout"})
		}
		logger.LogHandlerError(c, "auth.logout", status, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	userID := c.GetString("userID")
	user, err := h.service.Me(c.Request.Context(), userID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrUserNotFound):
			status = http.StatusUnauthorized
			c.JSON(status, gin.H{"error": "user not found"})
		default:
			c.JSON(status, gin.H{"error": "could not fetch user"})
		}
		logger.LogHandlerError(c, "auth.me", status, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MeResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		logger.LogHandlerError(c, "auth.forgot_password", http.StatusInternalServerError, err)
	}
	// Same answer whether or not the account exists.
	c.Status(http.StatusNoContent)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		if errors.Is(err, ErrInvalidResetCode) {
			httpx.Error(c, http.StatusBadRequest, "invalid or expired reset code")
			return
		}
		logger.LogHandlerError(c, "auth.reset_password", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not reset password")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), req.CurrentPassword, req.NewPassword)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWrongPassword):
		httpx.Error(c, http.StatusForbidden, "current password is incorrect")
	default:
		logger.LogHandlerError(c, "auth.change_password", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not change password")
	}
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWrongPassword):
		httpx.Error(c, http.StatusForbidden, "password is incorrect")
	default:
		logger.LogHandlerError(c, "auth.delete_account", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not delete account")
	}
}
