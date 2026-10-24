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
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: toMeResponse(user)})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, "auth.register", &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			httpx.Fail(c, "auth.register", http.StatusConflict, "email already exists", err)
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, "auth.register", http.StatusBadRequest, "password must be 8 to 72 characters", err)
		case errors.Is(err, ErrInvalidEmail):
			httpx.Fail(c, "auth.register", http.StatusBadRequest, "invalid email", err)
		default:
			httpx.Internal(c, "auth.register", err)
		}
		return
	}
	writeAuth(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, "auth.login", &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, "auth.login", http.StatusUnauthorized, "invalid credentials", err)
		case errors.Is(err, ErrAccountSuspended):
			httpx.Fail(c, "auth.login", http.StatusForbidden, "account suspended", err)
		default:
			httpx.Internal(c, "auth.login", err)
		}
		return
	}
	writeAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, "auth.refresh", &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken):
			httpx.Fail(c, "auth.refresh", http.StatusUnauthorized, "invalid refresh token", err)
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, "auth.refresh", http.StatusUnauthorized, "user not found", err)
		default:
			httpx.Internal(c, "auth.refresh", err)
		}
		return
	}
	writeAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, "auth.logout", &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.Fail(c, "auth.logout", http.StatusBadRequest, "invalid refresh token", err)
			return
		}
		httpx.Internal(c, "auth.logout", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, "auth.me", http.StatusUnauthorized, "user not found", err)
			return
		}
		httpx.Internal(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, "auth.forgot", &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Internal(c, "auth.forgot", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, "auth.reset", &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, ErrInvalidResetCode):
			httpx.Fail(c, "auth.reset", http.StatusBadRequest, "invalid or expired code", err)
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, "auth.reset", http.StatusBadRequest, "password must be 8 to 72 characters", err)
		default:
			httpx.Internal(c, "auth.reset", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, "auth.delete", &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, "auth.delete", http.StatusForbidden, "invalid password", err)
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, "auth.delete", http.StatusUnauthorized, "user not found", err)
		default:
			httpx.Internal(c, "auth.delete", err)
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}
