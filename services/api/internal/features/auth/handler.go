package auth

import (
	"errors"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service      *Service
	loginLimiter *middleware.Limiter
	resetLimiter *middleware.Limiter
}

func NewHandler(service *Service, loginLimiter, resetLimiter *middleware.Limiter) *Handler {
	return &Handler{service: service, loginLimiter: loginLimiter, resetLimiter: resetLimiter}
}

func respondAuth(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         UserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	})
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, ErrEmailExists):
		return platformerrors.Conflict("email_exists", "email already exists")
	case errors.Is(err, ErrInvalidCredentials):
		return platformerrors.Unauthorized("invalid_credentials", "invalid credentials")
	case errors.Is(err, ErrInvalidRefreshToken):
		return platformerrors.Unauthorized("invalid_refresh_token", "invalid refresh token")
	case errors.Is(err, ErrUserNotFound):
		return platformerrors.Unauthorized("unauthorized", "user not found")
	case errors.Is(err, ErrAccountSuspended):
		return platformerrors.Forbidden("account_suspended", "this account has been suspended")
	}
	return err
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		platformerrors.Respond(c, mapAuthError(err))
		return
	}
	respondAuth(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	if ok, _ := h.loginLimiter.Allow("login:" + normalizeEmail(req.Email)); !ok {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later", "code": "rate_limited"})
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		platformerrors.Respond(c, mapAuthError(err))
		return
	}
	respondAuth(c, http.StatusOK, tokens, user)
	logger.LogHandlerEvent(c, "auth.login.success", http.StatusOK, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		platformerrors.Respond(c, mapAuthError(err))
		return
	}
	respondAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		platformerrors.Respond(c, mapAuthError(err))
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	if ok, _ := h.resetLimiter.Allow("forgot:" + normalizeEmail(req.Email)); ok {
		if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
			logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
		}
	}
	// Same answer whether or not the email exists, or sending failed.
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword); err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), c.GetString("userID"), req.Password); err != nil {
		platformerrors.Respond(c, err)
		return
	}
	logger.LogHandlerEvent(c, "auth.account.deleted", http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}
