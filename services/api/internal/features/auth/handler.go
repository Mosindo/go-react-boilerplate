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

func toAuthResponse(tokens Tokens, user User) AuthResponse {
	return AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         UserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEmail):
			httpx.BadRequest(c, "invalid email address")
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, "password must contain 8 to 72 characters")
		case errors.Is(err, ErrEmailExists):
			httpx.Fail(c, http.StatusConflict, httpx.CodeConflict, "email already exists")
		default:
			logger.LogHandlerError(c, "auth.register", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, toAuthResponse(tokens, user))
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid credentials")
			return
		}
		logger.LogHandlerError(c, "auth.login", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toAuthResponse(tokens, user))
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid refresh token")
			return
		}
		logger.LogHandlerError(c, "auth.refresh", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toAuthResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, &req) {
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

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		// Never tell the caller: the response must not depend on the account.
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "password must contain 8 to 72 characters")
	case errors.Is(err, ErrInvalidResetCode):
		httpx.Fail(c, http.StatusBadRequest, "invalid_code", "invalid or expired code")
	default:
		logger.LogHandlerError(c, "auth.reset", http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "password must contain 8 to 72 characters")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, httpx.CodeForbidden, "current password is incorrect")
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "user not found")
	default:
		logger.LogHandlerError(c, "auth.change_password", http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, httpx.CodeForbidden, "password is incorrect")
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "user not found")
	default:
		logger.LogHandlerError(c, "auth.delete_account", http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}
