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

func toMe(u User) MeResponse {
	return MeResponse{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt}
}

func respondTokens(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMe(user)})
}

func fail(c *gin.Context, op string, status int, message string, err error) {
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, gin.H{"error": message})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.register.bind", http.StatusBadRequest, "invalid request: valid email and password (8-72 characters) required", err)
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			fail(c, "auth.register", http.StatusConflict, "email already exists", err)
		case errors.Is(err, ErrWeakPassword):
			fail(c, "auth.register", http.StatusBadRequest, ErrWeakPassword.Error(), err)
		case errors.Is(err, ErrInvalidEmail):
			fail(c, "auth.register", http.StatusBadRequest, ErrInvalidEmail.Error(), err)
		default:
			fail(c, "auth.register", http.StatusInternalServerError, "could not create user", err)
		}
		return
	}
	respondTokens(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.login.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			fail(c, "auth.login", http.StatusUnauthorized, "invalid credentials", err)
		case errors.Is(err, ErrAccountSuspended):
			fail(c, "auth.login", http.StatusForbidden, "this account is suspended", err)
		default:
			fail(c, "auth.login", http.StatusInternalServerError, "could not login", err)
		}
		return
	}
	respondTokens(c, http.StatusOK, tokens, user)
	logger.LogHandlerEvent(c, "auth.login.success", http.StatusOK, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.refresh.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
			fail(c, "auth.refresh", http.StatusUnauthorized, "invalid refresh token", err)
		case errors.Is(err, ErrAccountSuspended):
			fail(c, "auth.refresh", http.StatusForbidden, "this account is suspended", err)
		default:
			fail(c, "auth.refresh", http.StatusInternalServerError, "could not refresh session", err)
		}
		return
	}
	respondTokens(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.logout.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			fail(c, "auth.logout", http.StatusBadRequest, "invalid refresh token", err)
			return
		}
		fail(c, "auth.logout", http.StatusInternalServerError, "could not logout", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			fail(c, "auth.me", http.StatusUnauthorized, "user not found", err)
			return
		}
		fail(c, "auth.me", http.StatusInternalServerError, "could not fetch user", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMe(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.forgot.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		// Same response as success: delivery problems must not leak whether the email exists.
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if the account exists, a code was sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.reset.bind", http.StatusBadRequest, "invalid request: email, 8-digit code and new password required", err)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			fail(c, "auth.reset", http.StatusBadRequest, ErrWeakPassword.Error(), err)
		case errors.Is(err, ErrInvalidResetCode):
			fail(c, "auth.reset", http.StatusBadRequest, "invalid or expired code", err)
		default:
			fail(c, "auth.reset", http.StatusInternalServerError, "could not reset password", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.password.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			fail(c, "auth.password", http.StatusBadRequest, ErrWeakPassword.Error(), err)
		case errors.Is(err, ErrInvalidCredentials):
			fail(c, "auth.password", http.StatusForbidden, "current password is incorrect", err)
		case errors.Is(err, ErrUserNotFound):
			fail(c, "auth.password", http.StatusUnauthorized, "user not found", err)
		default:
			fail(c, "auth.password", http.StatusInternalServerError, "could not change password", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.delete.bind", http.StatusBadRequest, "password confirmation required", err)
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			fail(c, "auth.delete", http.StatusForbidden, "password is incorrect", err)
		case errors.Is(err, ErrUserNotFound):
			fail(c, "auth.delete", http.StatusUnauthorized, "user not found", err)
		default:
			fail(c, "auth.delete", http.StatusInternalServerError, "could not delete account", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}
