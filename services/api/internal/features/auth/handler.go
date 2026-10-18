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

func (h *Handler) authResponse(tokens Tokens, user User) AuthResponse {
	return AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toMeResponse(user),
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		status := http.StatusInternalServerError
		message := "could not create user"
		switch {
		case errors.Is(err, ErrEmailExists):
			status, message = http.StatusConflict, "email already exists"
		case errors.Is(err, ErrWeakPassword):
			status, message = http.StatusBadRequest, ErrWeakPassword.Error()
		case errors.Is(err, ErrInvalidEmail):
			status, message = http.StatusBadRequest, ErrInvalidEmail.Error()
		}
		logger.LogHandlerError(c, "auth.register", status, err)
		httpx.Error(c, status, message)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, h.authResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		status, message := http.StatusInternalServerError, "could not login"
		if errors.Is(err, ErrInvalidCredentials) {
			status, message = http.StatusUnauthorized, "invalid credentials"
		}
		logger.LogHandlerError(c, "auth.login", status, err)
		httpx.Error(c, status, message)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, h.authResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.login.success", http.StatusOK, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		status, message := http.StatusInternalServerError, "could not refresh session"
		switch {
		case errors.Is(err, ErrInvalidRefreshToken):
			status, message = http.StatusUnauthorized, "invalid refresh token"
		case errors.Is(err, ErrUserNotFound):
			status, message = http.StatusUnauthorized, "user not found"
		}
		logger.LogHandlerError(c, "auth.refresh", status, err)
		httpx.Error(c, status, message)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, h.authResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		status, message := http.StatusInternalServerError, "could not logout"
		if errors.Is(err, ErrInvalidRefreshToken) {
			status, message = http.StatusBadRequest, "invalid refresh token"
		}
		logger.LogHandlerError(c, "auth.logout", status, err)
		httpx.Error(c, status, message)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		status, message := http.StatusInternalServerError, "could not fetch user"
		if errors.Is(err, ErrUserNotFound) {
			status, message = http.StatusUnauthorized, "user not found"
		}
		logger.LogHandlerError(c, "auth.me", status, err)
		httpx.Error(c, status, message)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		// Logged but not surfaced: the response must not depend on the account.
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "if the address is registered, a code was sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		status, message := http.StatusInternalServerError, "could not reset password"
		switch {
		case errors.Is(err, ErrInvalidResetCode):
			status, message = http.StatusBadRequest, ErrInvalidResetCode.Error()
		case errors.Is(err, ErrWeakPassword):
			status, message = http.StatusBadRequest, ErrWeakPassword.Error()
		}
		logger.LogHandlerError(c, "auth.reset", status, err)
		httpx.Error(c, status, message)
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
	err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	if err != nil {
		status, message := http.StatusInternalServerError, "could not change password"
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			status, message = http.StatusForbidden, "current password is incorrect"
		case errors.Is(err, ErrWeakPassword):
			status, message = http.StatusBadRequest, ErrWeakPassword.Error()
		case errors.Is(err, ErrUserNotFound):
			status, message = http.StatusUnauthorized, "user not found"
		}
		logger.LogHandlerError(c, "auth.change_password", status, err)
		httpx.Error(c, status, message)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	userID := c.GetString("userID")
	if err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		status, message := http.StatusInternalServerError, "could not delete account"
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			status, message = http.StatusForbidden, "password is incorrect"
		case errors.Is(err, ErrUserNotFound):
			status, message = http.StatusUnauthorized, "user not found"
		}
		logger.LogHandlerError(c, "auth.delete_account", status, err)
		httpx.Error(c, status, message)
		return
	}
	logger.LogHandlerEvent(c, "auth.delete_account.success", http.StatusNoContent, map[string]string{"deleted_user_id": userID})
	c.Status(http.StatusNoContent)
}
