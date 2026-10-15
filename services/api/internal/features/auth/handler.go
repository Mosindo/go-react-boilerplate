package auth

import (
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
	if !httpx.Bind(c, &req) {
		return
	}

	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.register", err)
		return
	}

	httpx.NoStore(c)
	c.JSON(http.StatusCreated, toAuthResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.Bind(c, &req) {
		return
	}

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.login", err)
		return
	}

	httpx.NoStore(c)
	c.JSON(http.StatusOK, toAuthResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.login.success", http.StatusOK, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.Bind(c, &req) {
		return
	}

	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.refresh", err)
		return
	}

	httpx.NoStore(c)
	c.JSON(http.StatusOK, toAuthResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.Bind(c, &req) {
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Fail(c, "auth.logout", err)
		return
	}

	httpx.NoStore(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "auth.me", err)
		return
	}

	httpx.NoStore(c)
	c.JSON(http.StatusOK, toMeResponse(user))
}

// ForgotPassword always answers 202 so the endpoint cannot be used to discover accounts.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, "auth.forgot", err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		httpx.Fail(c, "auth.reset", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), req.CurrentPassword, req.NewPassword); err != nil {
		httpx.Fail(c, "auth.change_password", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.Bind(c, &req) {
		return
	}
	userID := httpx.UserID(c)
	if err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		httpx.Fail(c, "auth.delete_account", err)
		return
	}
	logger.LogHandlerEvent(c, "auth.account_deleted", http.StatusNoContent, map[string]string{"deleted_user_id": userID})
	c.Status(http.StatusNoContent)
}

func toMeResponse(user User) MeResponse {
	return MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}

func toAuthResponse(tokens Tokens, user User) AuthResponse {
	return AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toMeResponse(user),
	}
}
