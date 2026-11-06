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

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func toMe(u User) MeResponse { return MeResponse{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt} }

func respondAuth(c *gin.Context, status int, tokens Tokens, user User) {
	httpx.NoStore(c)
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMe(user)})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "a valid email and a password of 8 to 72 characters are required")
		return
	}
	if !ValidEmail(normalizeEmail(req.Email)) {
		httpx.BadRequest(c, "a valid email is required")
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrEmailExists):
		httpx.Error(c, http.StatusConflict, "email_taken", "email already exists")
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "password must be 8 to 72 characters")
	case err != nil:
		httpx.Internal(c, "auth.register", err)
	default:
		respondAuth(c, http.StatusCreated, tokens, user)
		logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "email and password are required")
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case err != nil:
		httpx.Internal(c, "auth.login", err)
	default:
		respondAuth(c, http.StatusOK, tokens, user)
	}
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "refreshToken is required")
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusUnauthorized, "invalid_refresh_token", "invalid refresh token")
	case err != nil:
		httpx.Internal(c, "auth.refresh", err)
	default:
		respondAuth(c, http.StatusOK, tokens, user)
	}
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "refreshToken is required")
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.BadRequest(c, "invalid refresh token")
			return
		}
		httpx.Internal(c, "auth.logout", err)
		return
	}
	httpx.NoStore(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "user not found")
	case err != nil:
		httpx.Internal(c, "auth.me", err)
	default:
		httpx.NoStore(c)
		c.JSON(http.StatusOK, toMe(user))
	}
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || !ValidEmail(normalizeEmail(req.Email)) {
		httpx.BadRequest(c, "a valid email is required")
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		// Log but answer identically so the endpoint cannot be used to probe accounts.
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if the account exists, a code was sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "email, code and a new password of 8 to 72 characters are required")
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword)
	switch {
	case errors.Is(err, ErrInvalidResetCode):
		httpx.Error(c, http.StatusBadRequest, "invalid_code", "invalid or expired code")
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "password must be 8 to 72 characters")
	case err != nil:
		httpx.Internal(c, "auth.reset", err)
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "current and new passwords are required")
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), req.CurrentPassword, req.NewPassword)
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusForbidden, "invalid_credentials", "current password is incorrect")
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "password must be 8 to 72 characters")
	case err != nil:
		httpx.Internal(c, "auth.change_password", err)
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "password confirmation is required")
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password)
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusForbidden, "invalid_credentials", "password is incorrect")
	case err != nil:
		httpx.Internal(c, "auth.delete_account", err)
	default:
		c.Status(http.StatusNoContent)
	}
}
