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
		httpx.BadRequest(c, "invalid_request", "a valid email and a password of 8 to 128 characters are required")
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEmail):
			httpx.BadRequest(c, "invalid_email", err.Error())
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, "weak_password", err.Error())
		case errors.Is(err, ErrEmailExists):
			httpx.Fail(c, http.StatusConflict, "email_exists", "an account already exists for this email")
		default:
			httpx.Internal(c, "auth.register", err)
		}
		return
	}
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
	writeAuth(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "email and password are required")
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
			return
		}
		httpx.Internal(c, "auth.login", err)
		return
	}
	writeAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "refresh token required")
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "session expired, please sign in again")
			return
		}
		httpx.Internal(c, "auth.refresh", err)
		return
	}
	writeAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "refresh token required")
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
		httpx.BadRequest(c, "invalid_request", "a valid email is required")
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Internal(c, "auth.forgot_password", err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if an account exists, a code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "email, 6-digit code and new password are required")
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, "weak_password", err.Error())
		case errors.Is(err, ErrInvalidResetCode):
			httpx.BadRequest(c, "invalid_reset_code", err.Error())
		default:
			httpx.Internal(c, "auth.reset_password", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "account not found")
			return
		}
		httpx.Internal(c, "auth.me", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "current and new password are required")
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, "weak_password", err.Error())
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, http.StatusForbidden, "invalid_password", "current password is incorrect")
		default:
			httpx.Internal(c, "auth.change_password", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "password confirmation required")
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, http.StatusForbidden, "invalid_password", "password is incorrect")
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "account not found")
		default:
			httpx.Internal(c, "auth.delete_account", err)
		}
		return
	}
	logger.LogHandlerEvent(c, "auth.account_deleted", http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}

func writeAuth(c *gin.Context, status int, tokens Tokens, user User) {
	httpx.NoStore(c)
	c.JSON(status, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toMeResponse(user),
	})
}

func toMeResponse(user User) MeResponse {
	return MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}
