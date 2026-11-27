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
	httpx.NoStore(c)
	c.JSON(status, AuthResponse{
		Token:        tokens.AccessToken,
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
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			httpx.Fail(c, http.StatusConflict, "email_exists", "Un compte existe déjà avec cet e-mail.", err)
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, "weak_password", "Le mot de passe doit contenir au moins 8 caractères, dont une lettre et un chiffre.")
		case errors.Is(err, ErrInvalidEmail):
			httpx.BadRequest(c, "invalid_email", "Adresse e-mail invalide.")
		default:
			httpx.Internal(c, err)
		}
		return
	}

	writeSession(c, http.StatusCreated, tokens, user)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "E-mail ou mot de passe incorrect.", err)
			return
		}
		httpx.Internal(c, err)
		return
	}

	writeSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) || errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "Session expirée, reconnecte-toi.", err)
			return
		}
		httpx.Internal(c, err)
		return
	}

	writeSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			httpx.BadRequest(c, "invalid_refresh_token", "Session invalide.")
			return
		}
		httpx.Internal(c, err)
		return
	}

	httpx.NoStore(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		// Still answer 202 so the response never reveals account existence.
		logger.LogHandlerError(c, "auth.forgot_password", http.StatusAccepted, err)
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			httpx.BadRequest(c, "weak_password", "Le mot de passe doit contenir au moins 8 caractères, dont une lettre et un chiffre.")
		case errors.Is(err, ErrInvalidResetCode):
			httpx.BadRequest(c, "invalid_reset_code", "Code invalide ou expiré.")
		default:
			httpx.Internal(c, err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, "user_not_found", "Compte introuvable.", err)
			return
		}
		httpx.Internal(c, err)
		return
	}

	httpx.NoStore(c)
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, http.StatusForbidden, "invalid_password", "Mot de passe incorrect.", err)
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, http.StatusUnauthorized, "user_not_found", "Compte introuvable.", err)
		default:
			httpx.Internal(c, err)
		}
		return
	}
	logger.LogHandlerEvent(c, "auth.account_deleted", http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}
