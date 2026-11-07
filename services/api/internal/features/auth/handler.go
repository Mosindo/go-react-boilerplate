package auth

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

// validEmail trims and lower-cases the address in place and checks it is a plain address (no display name).
func validEmail(email *string) bool {
	*email = normalizeEmail(*email)
	addr, err := mail.ParseAddress(*email)
	return err == nil && addr.Address == *email && strings.Contains(*email, ".")
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) authResponse(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid email or password (8-72 characters)")
		return
	}
	if !validEmail(&req.Email) {
		httpx.BadRequest(c, "invalid email or password (8-72 characters)")
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			httpx.Fail(c, http.StatusConflict, "email_exists", "email already exists")
			return
		}
		logger.LogHandlerError(c, "auth.register", 500, err)
		httpx.Internal(c)
		return
	}
	h.authResponse(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if !validEmail(&req.Email) {
		httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			return
		}
		logger.LogHandlerError(c, "auth.login", 500, err)
		httpx.Internal(c)
		return
	}
	h.authResponse(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "invalid refresh token")
			return
		}
		logger.LogHandlerError(c, "auth.refresh", 500, err)
		httpx.Internal(c)
		return
	}
	h.authResponse(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		logger.LogHandlerError(c, "auth.logout", 500, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		logger.LogHandlerError(c, "auth.me", 500, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt})
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid email")
		return
	}
	if !validEmail(&req.Email) {
		httpx.BadRequest(c, "invalid email")
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		logger.LogHandlerError(c, "auth.forgot", 500, err)
	}
	// Always 204: do not reveal whether the account exists.
	c.Status(http.StatusNoContent)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid token or password (8-72 characters)")
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, ErrInvalidResetToken) {
			httpx.Fail(c, http.StatusBadRequest, "invalid_reset_token", "invalid or expired reset token")
			return
		}
		logger.LogHandlerError(c, "auth.reset", 500, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "current password is incorrect")
			return
		}
		logger.LogHandlerError(c, "auth.change_password", 500, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "password confirmation required")
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "password is incorrect")
			return
		}
		logger.LogHandlerError(c, "auth.delete_account", 500, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
