package auth

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/authtoken"
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
	if !httpx.BindJSON(c, "auth.register", &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrEmailExists):
		httpx.Fail(c, http.StatusConflict, "email_taken", "an account already exists for this email")
		return
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error())
		return
	case errors.Is(err, ErrInvalidEmail):
		httpx.Fail(c, http.StatusBadRequest, "invalid_email", err.Error())
		return
	case err != nil:
		httpx.FailInternal(c, "auth.register", err)
		return
	}
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": user.ID})
	h.respondSession(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, "auth.login", &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	case err != nil:
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
	switch {
	case errors.Is(err, ErrInvalidRefreshToken), errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "session expired, please sign in again")
		return
	case err != nil:
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
			httpx.Fail(c, http.StatusBadRequest, "invalid_refresh_token", "invalid refresh token")
			return
		}
		httpx.FailInternal(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, "auth.forgot", &req) {
		return
	}
	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		httpx.FailInternal(c, "auth.forgot", err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "if an account exists, a reset code has been sent"})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, "auth.reset", &req) {
		return
	}
	tokens, user, err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password, c.Request.UserAgent(), c.ClientIP())
	switch {
	case errors.Is(err, ErrInvalidResetCode):
		httpx.Fail(c, http.StatusBadRequest, "invalid_reset_code", "invalid or expired code")
		return
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error())
		return
	case err != nil:
		httpx.FailInternal(c, "auth.reset", err)
		return
	}
	h.respondSession(c, http.StatusOK, tokens, user)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	case err != nil:
		httpx.FailInternal(c, "auth.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toMeResponse(user))
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !httpx.BindJSON(c, "auth.change_password", &req) {
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), httpx.UserID(c), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "current password is incorrect")
		return
	case errors.Is(err, ErrWeakPassword):
		httpx.Fail(c, http.StatusBadRequest, "weak_password", err.Error())
		return
	case err != nil:
		httpx.FailInternal(c, "auth.change_password", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, "auth.delete_account", &req) {
		return
	}
	userID := httpx.UserID(c)
	err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password)
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "password is incorrect")
		return
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "account not found")
		return
	case err != nil:
		httpx.FailInternal(c, "auth.delete_account", err)
		return
	}
	logger.LogHandlerEvent(c, "auth.account_deleted", http.StatusNoContent, map[string]string{"deleted_user_id": userID})
	c.Status(http.StatusNoContent)
}

func (h *Handler) RealtimeTicket(c *gin.Context) {
	ticket, err := h.service.RealtimeTicket(httpx.UserID(c), c.GetString("sessionID"))
	if err != nil {
		httpx.FailInternal(c, "auth.realtime_ticket", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, RealtimeTicketResponse{Ticket: ticket, ExpiresIn: int(authtoken.RealtimeTTL.Seconds())})
}
