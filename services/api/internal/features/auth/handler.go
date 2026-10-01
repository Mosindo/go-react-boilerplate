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

func authResponse(tokens Tokens, user User) AuthResponse {
	return AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "email ou mot de passe invalide (8 à 72 caractères)")
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), string(req.Email), req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrEmailExists):
		httpx.Error(c, http.StatusConflict, "email_exists", "un compte existe déjà avec cet email")
		return
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "le mot de passe doit contenir entre 8 et 72 caractères")
		return
	case err != nil:
		httpx.Internal(c, "auth.register", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, authResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "email ou mot de passe invalide")
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), string(req.Email), req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", "email ou mot de passe incorrect")
		return
	case err != nil:
		httpx.Internal(c, "auth.login", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "jeton de rafraîchissement invalide")
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusUnauthorized, "invalid_refresh_token", "session expirée, reconnectez-vous")
		return
	case err != nil:
		httpx.Internal(c, "auth.refresh", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "jeton de rafraîchissement invalide")
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.BadRequest(c, "jeton de rafraîchissement invalide")
			return
		}
		httpx.Internal(c, "auth.logout", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

// Forgot always answers 202 so the endpoint cannot be used to discover registered emails.
func (h *Handler) Forgot(c *gin.Context) {
	var req ForgotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "email invalide")
		return
	}
	h.service.RequestResetAsync(string(req.Email))
	c.Status(http.StatusAccepted)
}

func (h *Handler) Reset(c *gin.Context) {
	var req ResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "demande invalide (mot de passe de 8 à 72 caractères)")
		return
	}
	err := h.service.ResetPassword(c.Request.Context(), string(req.Email), req.Code, req.NewPassword)
	switch {
	case errors.Is(err, ErrInvalidResetCode):
		httpx.Error(c, http.StatusBadRequest, "invalid_code", "code invalide ou expiré")
		return
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "le mot de passe doit contenir entre 8 et 72 caractères")
		return
	case err != nil:
		httpx.Internal(c, "auth.reset", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if errors.Is(err, ErrUserNotFound) {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "compte introuvable")
		return
	}
	if err != nil {
		httpx.Internal(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt})
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "mot de passe requis")
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password)
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusForbidden, "invalid_credentials", "mot de passe incorrect")
		return
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "compte introuvable")
		return
	case err != nil:
		httpx.Internal(c, "auth.delete_account", err)
		return
	}
	c.Status(http.StatusNoContent)
}
