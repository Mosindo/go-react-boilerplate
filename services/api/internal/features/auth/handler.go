package auth

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
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
	if !httpx.BindJSON(c, &req) {
		return
	}
	t, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, AuthResponse{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, User: toMeResponse(user)})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	t, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, AuthResponse{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, User: toMeResponse(user)})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	t, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, AuthResponse{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, User: toMeResponse(user)})
}

func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Fail(c, http.StatusForbidden, "invalid_password", "Mot de passe incorrect.", nil)
			return
		}
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrEmailExists):
		httpx.Fail(c, http.StatusConflict, "email_taken", "Un compte existe déjà avec cet e-mail.", nil)
	case errors.Is(err, ErrInvalidEmail):
		httpx.BadRequest(c, "invalid_email", "Adresse e-mail invalide.")
	case errors.Is(err, ErrWeakPassword):
		httpx.BadRequest(c, "weak_password", "Le mot de passe doit contenir 8 à 72 caractères, dont au moins une lettre et un chiffre.")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "E-mail ou mot de passe incorrect.", nil)
	case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_session", "Session expirée, reconnecte-toi.", nil)
	case errors.Is(err, ErrInvalidResetCode):
		httpx.BadRequest(c, "invalid_code", "Code invalide ou expiré.")
	default:
		httpx.Internal(c, err)
	}
}
