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

func (h *Handler) respondSession(c *gin.Context, status int, tokens Tokens, user User) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toMeResponse(user),
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		h.respondSession(c, http.StatusCreated, tokens, user)
		logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
	case errors.Is(err, ErrInvalidEmail):
		httpx.BadRequest(c, "invalid_email", "adresse e-mail invalide")
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "weak_password", "le mot de passe doit contenir 8 à 72 caractères, dont au moins une lettre et un chiffre")
	case errors.Is(err, ErrEmailExists):
		httpx.Fail(c, http.StatusConflict, "email_taken", "un compte existe déjà avec cette adresse")
	default:
		httpx.Internal(c, "auth.register", err)
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		h.respondSession(c, http.StatusOK, tokens, user)
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "e-mail ou mot de passe incorrect")
	default:
		httpx.Internal(c, "auth.login", err)
	}
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	switch {
	case err == nil:
		h.respondSession(c, http.StatusOK, tokens, user)
	case errors.Is(err, ErrInvalidRefreshToken):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "session expirée, reconnectez-vous")
	default:
		httpx.Internal(c, "auth.refresh", err)
	}
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil && !errors.Is(err, ErrInvalidRefreshToken) {
		httpx.Internal(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, toMeResponse(user))
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_token", "session invalide")
	default:
		httpx.Internal(c, "auth.me", err)
	}
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	err := h.service.RequestPasswordReset(c.Request.Context(), req.Email)
	switch {
	case err == nil:
		c.JSON(http.StatusAccepted, gin.H{"status": "sent"})
	case errors.Is(err, ErrInvalidEmail):
		httpx.BadRequest(c, "invalid_email", "adresse e-mail invalide")
	default:
		httpx.Internal(c, "auth.forgot_password", err)
	}
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "weak_password", "le mot de passe doit contenir 8 à 72 caractères, dont au moins une lettre et un chiffre")
	case errors.Is(err, ErrInvalidResetCode):
		httpx.BadRequest(c, "invalid_code", "code invalide ou expiré")
	default:
		httpx.Internal(c, "auth.reset_password", err)
	}
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	userID := httpx.UserID(c)
	err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password)
	switch {
	case err == nil:
		logger.LogHandlerEvent(c, "auth.account_deleted", http.StatusNoContent, map[string]string{"deleted_user_id": userID})
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, "invalid_password", "mot de passe incorrect")
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_token", "session invalide")
	default:
		httpx.Internal(c, "auth.delete_account", err)
	}
}
