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
	return MeResponse{
		ID:             user.ID,
		Email:          user.Email,
		OrganizationID: user.OrganizationID,
		CreatedAt:      user.CreatedAt,
	}
}

func writeSession(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toMe(user),
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "auth.register.bind", err)
		return
	}

	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			httpx.Fail(c, http.StatusConflict, "email_exists", "email already exists", "auth.register", err)
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error(), "auth.register", err)
		case errors.Is(err, ErrInvalidEmail):
			httpx.Fail(c, http.StatusBadRequest, "invalid_email", err.Error(), "auth.register", err)
		default:
			httpx.Fail(c, http.StatusInternalServerError, "internal", "could not create user", "auth.register", err)
		}
		return
	}

	writeSession(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "auth.login.bind", err)
		return
	}

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials", "auth.login", err)
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not login", "auth.login", err)
		return
	}

	writeSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "auth.refresh.bind", err)
		return
	}

	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "invalid refresh token", "auth.refresh", err)
		default:
			httpx.Fail(c, http.StatusInternalServerError, "internal", "could not refresh session", "auth.refresh", err)
		}
		return
	}

	writeSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "auth.logout.bind", err)
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.Fail(c, http.StatusBadRequest, "invalid_refresh_token", "invalid refresh token", "auth.logout", err)
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not logout", "auth.logout", err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "auth.forgot.bind", err)
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		// Still answer 202: the outcome must not reveal whether the account exists.
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if an account exists, a code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "auth.reset.bind", err)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error(), "auth.reset", err)
		case errors.Is(err, ErrInvalidResetCode):
			httpx.Fail(c, http.StatusBadRequest, "invalid_reset_code", "invalid or expired code", "auth.reset", err)
		default:
			httpx.Fail(c, http.StatusInternalServerError, "internal", "could not reset password", "auth.reset", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "auth.change_password.bind", err)
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "current password is incorrect", "auth.change_password", err)
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error(), "auth.change_password", err)
		default:
			httpx.Fail(c, http.StatusInternalServerError, "internal", "could not change password", "auth.change_password", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "user not found", "auth.me", err)
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not fetch user", "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMe(user))
}
