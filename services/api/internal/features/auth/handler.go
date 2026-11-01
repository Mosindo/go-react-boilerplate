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

func (h *Handler) respondTokens(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMe(user)})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if !validEmail(normalizeEmail(req.Email)) {
		httpx.Error(c, http.StatusBadRequest, "invalid email")
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrWeakPassword):
		httpx.Error(c, http.StatusBadRequest, "password must be 8-72 characters with letters and digits")
	case errors.Is(err, ErrEmailExists):
		httpx.Error(c, http.StatusConflict, "email already exists")
	case err != nil:
		logger.LogHandlerError(c, "auth.register", http.StatusInternalServerError, err)
		httpx.Internal(c)
	default:
		h.respondTokens(c, http.StatusCreated, tokens, user)
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if !validEmail(normalizeEmail(req.Email)) {
		httpx.Error(c, http.StatusBadRequest, "invalid email")
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, "invalid credentials")
	case err != nil:
		logger.LogHandlerError(c, "auth.login", http.StatusInternalServerError, err)
		httpx.Internal(c)
	default:
		h.respondTokens(c, http.StatusOK, tokens, user)
	}
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusUnauthorized, "invalid refresh token")
	case err != nil:
		logger.LogHandlerError(c, "auth.refresh", http.StatusInternalServerError, err)
		httpx.Internal(c)
	default:
		h.respondTokens(c, http.StatusOK, tokens, user)
	}
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		logger.LogHandlerError(c, "auth.logout", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if !validEmail(normalizeEmail(req.Email)) {
		httpx.Error(c, http.StatusBadRequest, "invalid email")
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		logger.LogHandlerError(c, "auth.forgot", http.StatusInternalServerError, err)
	}
	c.Status(http.StatusNoContent) // toujours 204: pas d'enumeration de comptes
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), req.Token, req.Password)
	switch {
	case errors.Is(err, ErrWeakPassword):
		httpx.Error(c, http.StatusBadRequest, "password must be 8-72 characters with letters and digits")
	case errors.Is(err, ErrInvalidResetToken):
		httpx.Error(c, http.StatusBadRequest, "invalid or expired reset token")
	case err != nil:
		logger.LogHandlerError(c, "auth.reset", http.StatusInternalServerError, err)
		httpx.Internal(c)
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
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
	c.JSON(http.StatusOK, toMe(user))
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password)
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusForbidden, "invalid password")
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusUnauthorized, "user not found")
	case err != nil:
		logger.LogHandlerError(c, "auth.delete", http.StatusInternalServerError, err)
		httpx.Internal(c)
	default:
		c.Status(http.StatusNoContent)
	}
}
