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
			c.JSON(status, gin.H{"error": "email already exists", "code": "email_exists"})
		case errors.Is(err, ErrWeakPassword):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": err.Error(), "code": "weak_password"})
		case errors.Is(err, ErrInvalidEmail):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": err.Error(), "code": "invalid_email"})
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
			ID:             user.ID,
			Email:          user.Email,
			OrganizationID: user.OrganizationID,
			CreatedAt:      user.CreatedAt,
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
			ID:             user.ID,
			Email:          user.Email,
			OrganizationID: user.OrganizationID,
			CreatedAt:      user.CreatedAt,
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
			ID:             user.ID,
			Email:          user.Email,
			OrganizationID: user.OrganizationID,
			CreatedAt:      user.CreatedAt,
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
		ID:             user.ID,
		Email:          user.Email,
		OrganizationID: user.OrganizationID,
		CreatedAt:      user.CreatedAt,
	})
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_email", "a valid email is required")
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Internal(c, "auth.forgot_password", err)
		return
	}
	// Same answer whether or not the account exists.
	c.JSON(http.StatusAccepted, gin.H{"status": "if this email is registered, a code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "email, 6-digit code and new password are required")
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password); err != nil {
		h.failAccount(c, "auth.reset_password", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "current and new passwords are required")
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword); err != nil {
		h.failAccount(c, "auth.change_password", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "password confirmation is required")
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		h.failAccount(c, "auth.delete_account", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) failAccount(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, op, http.StatusBadRequest, "weak_password", err.Error(), nil)
	case errors.Is(err, ErrInvalidResetCode):
		httpx.Fail(c, op, http.StatusBadRequest, "invalid_code", err.Error(), nil)
	case errors.Is(err, ErrWrongPassword):
		httpx.Fail(c, op, http.StatusForbidden, "wrong_password", err.Error(), nil)
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, op, http.StatusUnauthorized, "user_not_found", "user not found", nil)
	default:
		httpx.Internal(c, op, err)
	}
}
