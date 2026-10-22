package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service       *Service
	loginAttempts *middleware.RateLimiter
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service, loginAttempts: middleware.NewRateLimiter(10, 15*time.Minute)}
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
	return MeResponse{ID: user.ID, Email: user.Email, ProfileComplete: user.ProfileComplete, CreatedAt: user.CreatedAt}
}

// writeError maps service errors to HTTP responses. It returns the status used.
func writeError(c *gin.Context, op string, err error) {
	var validation *platformerrors.ValidationError
	status := http.StatusInternalServerError
	body := gin.H{"error": "internal error"}
	switch {
	case errors.As(err, &validation):
		status = http.StatusBadRequest
		body = gin.H{"error": validation.Message, "field": validation.Field}
	case errors.Is(err, ErrEmailExists):
		status = http.StatusConflict
		body = gin.H{"error": "email already exists"}
	case errors.Is(err, ErrInvalidCredentials):
		status = http.StatusUnauthorized
		body = gin.H{"error": "invalid credentials"}
	case errors.Is(err, ErrAccountSuspended):
		status = http.StatusForbidden
		body = gin.H{"error": "account suspended"}
	case errors.Is(err, ErrInvalidRefreshToken):
		status = http.StatusUnauthorized
		body = gin.H{"error": "invalid refresh token"}
	case errors.Is(err, ErrInvalidResetToken):
		status = http.StatusBadRequest
		body = gin.H{"error": "invalid or expired reset token"}
	case errors.Is(err, ErrUserNotFound):
		status = http.StatusUnauthorized
		body = gin.H{"error": "user not found"}
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, body)
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		writeError(c, "auth.register", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, authResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if ok, _ := h.loginAttempts.Allow("login:" + strings.ToLower(strings.TrimSpace(req.Email))); !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		writeError(c, "auth.login", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.login.success", http.StatusOK, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		writeError(c, "auth.refresh", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		writeError(c, "auth.logout", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeError(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, meResponse(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	// Same answer whether or not the account exists.
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		writeError(c, "auth.reset", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		writeError(c, "auth.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}
