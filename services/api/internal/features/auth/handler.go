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
		ExpiresIn:    int(AccessTokenTTL.Seconds()),
		User:         toMeResponse(user),
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, "auth.register", &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			httpx.Fail(c, http.StatusConflict, "email_taken", "Un compte existe déjà avec cet e-mail.")
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, http.StatusBadRequest, "weak_password", "Le mot de passe doit contenir 8 à 72 caractères, dont au moins une lettre et un chiffre.")
		case errors.Is(err, ErrInvalidEmail):
			httpx.Fail(c, http.StatusBadRequest, "invalid_email", "Adresse e-mail invalide.")
		default:
			httpx.FailInternal(c, "auth.register", err)
		}
		return
	}
	h.respondSession(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, "auth.login", &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "E-mail ou mot de passe incorrect.")
			return
		}
		httpx.FailInternal(c, "auth.login", err)
		return
	}
	h.respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, "auth.refresh", &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "Session expirée. Reconnecte-toi.")
			return
		}
		httpx.FailInternal(c, "auth.refresh", err)
		return
	}
	h.respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, "auth.logout", &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.Fail(c, http.StatusBadRequest, "invalid_refresh_token", "Jeton invalide.")
			return
		}
		httpx.FailInternal(c, "auth.logout", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "user_not_found", "Compte introuvable.")
			return
		}
		httpx.FailInternal(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, "auth.forgot", &req) {
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		// Do not leak delivery failures; they are logged server-side.
		logger.LogHandlerError(c, "auth.forgot", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, "auth.reset", &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, http.StatusBadRequest, "weak_password", "Le mot de passe doit contenir 8 à 72 caractères, dont au moins une lettre et un chiffre.")
		case errors.Is(err, ErrInvalidResetCode):
			httpx.Fail(c, http.StatusBadRequest, "invalid_code", "Code invalide ou expiré.")
		default:
			httpx.FailInternal(c, "auth.reset", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.BindJSON(c, "auth.change_password", &req) {
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "Mot de passe actuel incorrect.")
		case errors.Is(err, ErrWeakPassword):
			httpx.Fail(c, http.StatusBadRequest, "weak_password", "Le mot de passe doit contenir 8 à 72 caractères, dont au moins une lettre et un chiffre.")
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, http.StatusUnauthorized, "user_not_found", "Compte introuvable.")
		default:
			httpx.FailInternal(c, "auth.change_password", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, "auth.delete_account", &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "Mot de passe incorrect.")
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, http.StatusUnauthorized, "user_not_found", "Compte introuvable.")
		default:
			httpx.FailInternal(c, "auth.delete_account", err)
		}
		return
	}
	logger.LogHandlerEvent(c, "auth.delete_account.success", http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}
