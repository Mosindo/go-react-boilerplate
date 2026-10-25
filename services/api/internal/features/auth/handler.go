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

func toMe(user User) MeResponse {
	return MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt, ProfileComplete: user.ProfileComplete}
}

func authResponse(tokens Tokens, user User) AuthResponse {
	return AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMe(user)}
}

func bind(c *gin.Context, op string, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		logger.LogHandlerError(c, op+".bind", http.StatusBadRequest, err)
		httpx.BadRequest(c, "invalid request")
		return false
	}
	return true
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !bind(c, "auth.register", &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), string(req.Email), req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			httpx.Fail(c, http.StatusConflict, "email_taken", "email already exists")
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error())
		default:
			logger.LogHandlerError(c, "auth.register", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, authResponse(tokens, user))
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !bind(c, "auth.login", &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), string(req.Email), req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			return
		}
		logger.LogHandlerError(c, "auth.login", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !bind(c, "auth.refresh", &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "invalid refresh token")
			return
		}
		logger.LogHandlerError(c, "auth.refresh", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !bind(c, "auth.logout", &req) {
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
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		logger.LogHandlerError(c, "auth.me", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMe(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !bind(c, "auth.forgot", &req) {
		return
	}
	h.service.ForgotPassword(c.Request.Context(), string(req.Email))
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !bind(c, "auth.reset", &req) {
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), string(req.Email), req.Code, req.Password)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error())
	case errors.Is(err, ErrInvalidResetCode):
		httpx.Fail(c, http.StatusBadRequest, "invalid_code", err.Error())
	default:
		logger.LogHandlerError(c, "auth.reset", http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !bind(c, "me.password", &req) {
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error())
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "current password is incorrect")
	default:
		logger.LogHandlerError(c, "me.password", http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !bind(c, "me.delete", &req) {
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "password is incorrect")
	default:
		logger.LogHandlerError(c, "me.delete", http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}
