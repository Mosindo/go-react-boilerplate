package auth

import (
	"errors"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
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
		User:         toMe(user),
	}
}

func toMe(user User) MeResponse {
	return MeResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}

func badRequest(c *gin.Context, operation string, err error) {
	logger.LogHandlerError(c, operation, http.StatusBadRequest, err)
	platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", "invalid request")
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "auth.register.bind", err)
		return
	}

	email, ok := cleanEmail(req.Email)
	if !ok {
		badRequest(c, "auth.register.email", ErrInvalidCredentials)
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			logger.LogHandlerError(c, "auth.register", http.StatusConflict, err)
			platformerrors.Respond(c, http.StatusConflict, "email_taken", "email already exists")
			return
		}
		logger.LogHandlerError(c, "auth.register", http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not create user")
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, authResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "auth.login.bind", err)
		return
	}

	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			logger.LogHandlerError(c, "auth.login", http.StatusUnauthorized, err)
			platformerrors.Respond(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			return
		}
		logger.LogHandlerError(c, "auth.login", http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not login")
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
	logger.LogHandlerEvent(c, "auth.login.success", http.StatusOK, map[string]string{"authenticated_user_id": user.ID})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "auth.refresh.bind", err)
		return
	}

	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
			logger.LogHandlerError(c, "auth.refresh", http.StatusUnauthorized, err)
			platformerrors.Respond(c, http.StatusUnauthorized, "invalid_refresh_token", "invalid refresh token")
		default:
			logger.LogHandlerError(c, "auth.refresh", http.StatusInternalServerError, err)
			platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not refresh session")
		}
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "auth.logout.bind", err)
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			platformerrors.Respond(c, http.StatusBadRequest, "invalid_refresh_token", "invalid refresh token")
			return
		}
		logger.LogHandlerError(c, "auth.logout", http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not logout")
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			platformerrors.Respond(c, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		logger.LogHandlerError(c, "auth.me", http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not fetch user")
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMe(user))
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "auth.forgot.bind", err)
		return
	}
	email, ok := cleanEmail(req.Email)
	if !ok {
		badRequest(c, "auth.forgot.email", ErrInvalidCredentials)
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), email); err != nil {
		// The error is logged but not surfaced: the response must be identical
		// whether or not the email exists.
		logger.LogHandlerError(c, "auth.forgot", http.StatusInternalServerError, err)
	}
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "auth.reset.bind", err)
		return
	}
	email, ok := cleanEmail(req.Email)
	if !ok {
		badRequest(c, "auth.reset.email", ErrInvalidCredentials)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), email, req.Code, req.NewPassword); err != nil {
		if errors.Is(err, ErrInvalidResetCode) {
			platformerrors.Respond(c, http.StatusBadRequest, "invalid_code", "invalid or expired recovery code")
			return
		}
		logger.LogHandlerError(c, "auth.reset", http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not reset password")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "auth.delete.bind", err)
		return
	}
	userID := c.GetString("userID")
	if err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			platformerrors.Respond(c, http.StatusForbidden, "invalid_credentials", "password is incorrect")
		case errors.Is(err, ErrUserNotFound):
			platformerrors.Respond(c, http.StatusNotFound, "not_found", "user not found")
		default:
			logger.LogHandlerError(c, "auth.delete", http.StatusInternalServerError, err)
			platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not delete account")
		}
		return
	}
	logger.LogHandlerEvent(c, "auth.delete.success", http.StatusNoContent, map[string]string{"deleted_user_id": userID})
	c.Status(http.StatusNoContent)
}
