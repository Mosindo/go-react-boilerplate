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

func writeAuth(c *gin.Context, status int, tokens Tokens, user User) {
	httpx.NoStore(c)
	c.JSON(status, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toMeResponse(user),
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.register.bind", "invalid request", err)
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, req.BirthDate, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		writeAuth(c, http.StatusCreated, tokens, user)
		logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
	case errors.Is(err, ErrEmailExists):
		httpx.Fail(c, "auth.register", http.StatusConflict, "email already exists", err)
	case errors.Is(err, ErrUnderage):
		httpx.Fail(c, "auth.register", http.StatusForbidden, "you must be at least 18 years old", err)
	case errors.Is(err, ErrInvalidBirthDate):
		httpx.BadRequest(c, "auth.register", "invalid birth date", err)
	default:
		httpx.Internal(c, "auth.register", err)
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.login.bind", "invalid request", err)
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		writeAuth(c, http.StatusOK, tokens, user)
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, "auth.login", http.StatusUnauthorized, "invalid credentials", err)
	default:
		httpx.Internal(c, "auth.login", err)
	}
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.refresh.bind", "invalid request", err)
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		writeAuth(c, http.StatusOK, tokens, user)
	case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, "auth.refresh", http.StatusUnauthorized, "invalid refresh token", err)
	default:
		httpx.Internal(c, "auth.refresh", err)
	}
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.logout.bind", "invalid request", err)
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.BadRequest(c, "auth.logout", "invalid refresh token", err)
			return
		}
		httpx.Internal(c, "auth.logout", err)
		return
	}
	httpx.NoStore(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	switch {
	case err == nil:
		httpx.NoStore(c)
		c.JSON(http.StatusOK, toMeResponse(user))
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, "auth.me", http.StatusUnauthorized, "user not found", err)
	default:
		httpx.Internal(c, "auth.me", err)
	}
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.forgot.bind", "invalid request", err)
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		// Do not leak whether the account exists: log, answer the same way.
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.reset.bind", "invalid request", err)
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInvalidResetCode):
		httpx.BadRequest(c, "auth.reset", "invalid or expired code", err)
	default:
		httpx.Internal(c, "auth.reset", err)
	}
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.delete.bind", "invalid request", err)
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password)
	switch {
	case err == nil:
		logger.LogHandlerEvent(c, "auth.delete.success", http.StatusNoContent, nil)
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, "auth.delete", http.StatusForbidden, "wrong password", err)
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, "auth.delete", http.StatusUnauthorized, "user not found", err)
	default:
		httpx.Internal(c, "auth.delete", err)
	}
}
