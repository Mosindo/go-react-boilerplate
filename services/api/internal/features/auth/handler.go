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

func toMeResponse(user User) MeResponse {
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
		User:         toMeResponse(user),
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.register.bind", "enter a valid e-mail and a password of 8 to 72 characters", err)
		return
	}

	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			httpx.FailCode(c, "auth.register", http.StatusConflict, "email_exists", "an account already exists for this e-mail", err)
			return
		}
		if errors.Is(err, ErrInvalidEmail) {
			httpx.FailCode(c, "auth.register", http.StatusBadRequest, "validation", "enter a valid e-mail address", err)
			return
		}
		httpx.Fail(c, "auth.register", http.StatusInternalServerError, "could not create account", err)
		return
	}

	writeSession(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.login.bind", "invalid request", err)
		return
	}

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.FailCode(c, "auth.login", http.StatusUnauthorized, "invalid_credentials", "invalid credentials", err)
			return
		}
		httpx.Fail(c, "auth.login", http.StatusInternalServerError, "could not login", err)
		return
	}

	writeSession(c, http.StatusOK, tokens, user)
	logger.LogHandlerEvent(c, "auth.login.success", http.StatusOK, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.refresh.bind", "invalid request", err)
		return
	}

	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.FailCode(c, "auth.refresh", http.StatusUnauthorized, "unauthenticated", "invalid refresh token", err)
			return
		}
		httpx.Fail(c, "auth.refresh", http.StatusInternalServerError, "could not refresh session", err)
		return
	}

	writeSession(c, http.StatusOK, tokens, user)
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
		httpx.Fail(c, "auth.logout", http.StatusInternalServerError, "could not logout", err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.FailCode(c, "auth.me", http.StatusUnauthorized, "unauthenticated", "user not found", err)
			return
		}
		httpx.Fail(c, "auth.me", http.StatusInternalServerError, "could not fetch user", err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.delete.bind", "password confirmation required", err)
		return
	}

	userID := httpx.UserID(c)
	if err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.FailCode(c, "auth.delete", http.StatusForbidden, "invalid_credentials", "incorrect password", err)
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, "auth.delete", http.StatusNotFound, "user not found", err)
		default:
			httpx.Fail(c, "auth.delete", http.StatusInternalServerError, "could not delete account", err)
		}
		return
	}

	logger.LogHandlerEvent(c, "auth.delete.success", http.StatusNoContent, map[string]string{"deleted_user_id": userID})
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.forgot.bind", "enter a valid e-mail", err)
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, "auth.forgot", http.StatusInternalServerError, "could not send reset code", err)
		return
	}
	// Same answer whether or not the account exists.
	c.JSON(http.StatusAccepted, gin.H{"status": "if an account exists, a code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "auth.reset.bind", "enter the 6-digit code and a password of 8 to 72 characters", err)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		if errors.Is(err, ErrInvalidResetCode) {
			httpx.FailCode(c, "auth.reset", http.StatusBadRequest, "invalid_code", "invalid or expired code", err)
			return
		}
		httpx.Fail(c, "auth.reset", http.StatusInternalServerError, "could not reset password", err)
		return
	}
	c.Status(http.StatusNoContent)
}
