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

func writeAuth(c *gin.Context, status int, tokens Tokens, user User) {
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
		httpx.BadRequest(c, "invalid_request", "a valid email and a password are required")
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
			httpx.Fail(c, http.StatusConflict, "email_exists", "an account already exists for this email", "auth.register", err)
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
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "email and password are required")
		return
	}

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password", "auth.login", err)
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
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "session expired, please sign in again", "auth.refresh", err)
			return
		}
		httpx.Internal(c, "auth.refresh", err)
		return
	}

	writeAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "refresh token required")
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.BadRequest(c, "invalid_refresh_token", "invalid refresh token")
			return
		}
		httpx.Internal(c, "auth.logout", err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "user not found", "auth.me", err)
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
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "a valid email is required")
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		httpx.Internal(c, "auth.password.forgot", err)
		return
	}
	// Always the same answer, whether or not the account exists.
	c.JSON(http.StatusAccepted, gin.H{"status": "if an account exists, a code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "email, code and new password are required")
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, "weak_password", err.Error())
		case errors.Is(err, ErrInvalidResetCode):
			httpx.BadRequest(c, "invalid_reset_code", err.Error())
		default:
			httpx.Internal(c, "auth.password.reset", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
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
		case errors.Is(err, ErrWrongPassword):
			httpx.Fail(c, http.StatusForbidden, "wrong_password", "current password is incorrect", "auth.password.change", nil)
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, "weak_password", err.Error())
		default:
			httpx.Internal(c, "auth.password.change", err)
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
	userID := httpx.UserID(c)
	if err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrWrongPassword):
			httpx.Fail(c, http.StatusForbidden, "wrong_password", "password is incorrect", "auth.delete", nil)
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "user not found", "auth.delete", err)
		default:
			httpx.Internal(c, "auth.delete", err)
		}
		return
	}
	logger.LogHandlerEvent(c, "auth.delete.success", http.StatusNoContent, map[string]string{"deleted_user_id": userID})
	c.Status(http.StatusNoContent)
}
