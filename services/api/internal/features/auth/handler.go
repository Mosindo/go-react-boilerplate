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
		httpx.BadRequest(c, "a valid email and a password of 8 to 72 characters are required")
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidEmail):
		httpx.BadRequest(c, err.Error())
		return
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error())
		return
	case errors.Is(err, ErrEmailExists):
		httpx.Fail(c, http.StatusConflict, "email_taken", "an account already exists for this email")
		return
	case err != nil:
		httpx.Internal(c, "auth.register", err)
		return
	}
	h.writeSession(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
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
		httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "incorrect email or password")
		return
	case err != nil:
		httpx.Internal(c, "auth.login", err)
		return
	}
	h.writeSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "refreshToken is required")
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidRefreshToken):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "your session has expired, please sign in again")
		return
	case err != nil:
		httpx.Internal(c, "auth.refresh", err)
		return
	}
	h.writeSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "refreshToken is required")
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Internal(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "a valid email is required")
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		httpx.Internal(c, "auth.forgot", err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if an account exists, a code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "email, 6-digit code and a new password of 8 to 72 characters are required")
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword)
	switch {
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error())
		return
	case errors.Is(err, ErrInvalidResetCode):
		httpx.Fail(c, http.StatusBadRequest, "invalid_code", "this code is invalid or has expired")
		return
	case err != nil:
		httpx.Internal(c, "auth.reset", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "account not found")
		return
	case err != nil:
		httpx.Internal(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMeResponse(user))
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
		httpx.Fail(c, http.StatusForbidden, "invalid_password", "incorrect password")
		return
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "account not found")
		return
	case err != nil:
		httpx.Internal(c, "auth.delete", err)
		return
	}
	logger.LogHandlerEvent(c, "auth.account_deleted", http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}

func (h *Handler) writeSession(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toMeResponse(user),
	})
}
