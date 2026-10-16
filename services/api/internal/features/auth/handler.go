package auth

import (
	"errors"
	"net/http"

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
		case errors.Is(err, ErrInvalidPassword):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "password must be between 8 and 72 characters"})
		case errors.Is(err, ErrInvalidEmail):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "invalid email address"})
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

// ForgotPassword always answers 202 so the endpoint cannot be used to enumerate accounts.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "auth.forgot.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusAccepted, gin.H{"status": "if the account exists, a code was sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "auth.reset.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrInvalidResetCode):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "invalid or expired code"})
		case errors.Is(err, ErrInvalidPassword):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "password must be between 8 and 72 characters"})
		default:
			c.JSON(status, gin.H{"error": "could not reset password"})
		}
		logger.LogHandlerError(c, "auth.reset", status, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}
