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
		httpx.BadRequest(c, "a valid email and a password of 8 to 72 characters are required")
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			httpx.Error(c, http.StatusConflict, "email already exists")
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, err.Error())
		default:
			logger.LogHandlerError(c, "auth.register", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	respondSession(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "email and password are required")
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Error(c, http.StatusUnauthorized, "invalid credentials")
		case errors.Is(err, ErrAccountSuspended):
			httpx.Error(c, http.StatusForbidden, "account suspended")
		default:
			logger.LogHandlerError(c, "auth.login", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "refreshToken is required")
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
			httpx.Error(c, http.StatusUnauthorized, "invalid refresh token")
		case errors.Is(err, ErrAccountSuspended):
			httpx.Error(c, http.StatusForbidden, "account suspended")
		default:
			logger.LogHandlerError(c, "auth.refresh", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "refreshToken is required")
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
	httpx.NoStore(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			httpx.Error(c, http.StatusUnauthorized, "user not found")
		case errors.Is(err, ErrAccountSuspended):
			httpx.Error(c, http.StatusForbidden, "account suspended")
		default:
			logger.LogHandlerError(c, "auth.me", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, toMe(user))
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "password is required to delete the account")
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Error(c, http.StatusForbidden, "incorrect password")
		case errors.Is(err, ErrUserNotFound):
			httpx.Error(c, http.StatusNotFound, "user not found")
		default:
			logger.LogHandlerError(c, "auth.delete", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	httpx.NoStore(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "a valid email is required")
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		// Logged but not surfaced: the response must not depend on the account existing.
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if the account exists, a recovery code was sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "email, code and a new password of 8 to 72 characters are required")
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, ErrInvalidResetCode):
			httpx.BadRequest(c, "invalid or expired code")
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, err.Error())
		default:
			logger.LogHandlerError(c, "auth.reset", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	httpx.NoStore(c)
	c.Status(http.StatusNoContent)
}

func respondSession(c *gin.Context, status int, tokens Tokens, user User) {
	httpx.NoStore(c)
	c.JSON(status, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toMe(user),
	})
}

func toMe(user User) MeResponse {
	return MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}
